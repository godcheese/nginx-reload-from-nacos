package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/util"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	version              = "1.0.0"
	firstInit            = true
	configFile           string
	config               *Config
	help                 bool
	mutex                sync.Mutex
	subscribeServicesMap map[string]SubscribeService
)

func main() {
	// 监听退出
	watchExit()

	// 帮助
	helpUsage()

	// 初始化 config.yaml 配置
	config = initConfig(configFile)

	subscribeServicesMap = getSubscribeServicesMap(config.Nacos.Discovery.SubscribeServices)

	// nginx conf 文件夹不存在就创建
	nginxConfPath := config.Nginx.ConfPath
	isExists := fileIsExists(nginxConfPath)
	if isExists {
		err := os.RemoveAll(nginxConfPath)
		checkErr(err)
	}
	err := os.MkdirAll(nginxConfPath, 0777)
	checkErr(err)
	discovery := config.Nacos.Discovery
	namingClient := getNacosNamingClient(discovery)

	for _, subscribeService := range discovery.SubscribeServices {
		subscribeParam := &vo.SubscribeParam{
			ServiceName: subscribeService.ServiceName,
			GroupName:   discovery.GroupName,
			Clusters:    []string{""},
			SubscribeCallback: func(instances []model.Instance, err error) {
				mutex.Lock()
				checkErr(err)
				nacosSubscribeCallback(namingClient, discovery, instances, nginxConfPath)
				defer mutex.Unlock()
			},
		}
		err = namingClient.Subscribe(subscribeParam)
		checkErr(err)
	}

	// 首次加载
	if firstInit {
		reloadNginx("")
		msg := fmt.Sprintf(config.WeWork.Messages["first-init-success"])
		log.Printf("msg: %s", msg)
		// 发送企业微信通知
		sendMsgToWeWork(msg)
		firstInit = false
	}

	// 保持运行状态
	for {
		time.Sleep(time.Second)
	}
}

// watchExit 监听退出
func watchExit() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for sc := range signalChan {
			if sc == syscall.SIGHUP || sc == syscall.SIGINT || sc == syscall.SIGTERM || sc == syscall.SIGQUIT {
				os.Exit(0)
			}
		}
	}()
}

// helpUsage 帮助
func helpUsage() {
	flag.BoolVar(&help, "h", false, "help")
	flag.StringVar(&configFile, "c", "./config.yaml", "config file")
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
}

// nacosSubscribeCallback 订阅 nacos 服务回调
func nacosSubscribeCallback(namingClient naming_client.INamingClient, discovery Discovery, instances []model.Instance, nginxConfPath string) {

	// 获取 service addresses
	serviceAddressesMap := getServiceAddressesMap(namingClient, instances)

	// 生成 nginx conf 配置文件
	generateNginxConf(serviceAddressesMap, nginxConfPath)

	if !firstInit && len(instances) > 0 {
		var serviceNames []string
		for _, instance := range instances {
			instanceServiceName := getNacosInstanceServiceName(instance.ServiceName)
			serviceNames = append(serviceNames, instance.Ip+":"+strconv.FormatUint(instance.Port, 10)+"@"+instanceServiceName.ServiceName)
		}
		serviceNamesStr := strings.Join(serviceNames, ", ")
		reloadNginx(serviceNamesStr)
		msg := fmt.Sprintf(config.WeWork.Messages["nginx-reload-success"], serviceNamesStr)
		log.Printf("msg2: %s", msg)
		sendMsgToWeWork(msg)
	}
}

// getServiceAddressesMap 获取 service addresses
func getServiceAddressesMap(namingClient naming_client.INamingClient, instances []model.Instance) map[string][]ServiceAddress {
	serviceAddressesMap := map[string][]ServiceAddress{}
	var serviceAddresses []ServiceAddress
	if len(instances) > 0 {
		for _, instance := range instances {
			if instance.Healthy && instance.Enable {
				instanceServiceName := getNacosInstanceServiceName(instance.ServiceName)
				serviceAddresses = serviceAddressesMap[instanceServiceName.ServiceName]
				if serviceAddresses == nil {
					serviceAddresses = []ServiceAddress{}
				}
				serviceAddress := ServiceAddress{
					Ip:     instance.Ip,
					Port:   instance.Port,
					Weight: uint64(instance.Weight),
				}
				serviceAddresses = append(serviceAddresses, serviceAddress)
				serviceAddressesMap[instanceServiceName.ServiceName] = serviceAddresses
			}
		}
	}
	return serviceAddressesMap
}

// getNacosInstanceServiceName 获取 nacos instance service name
func getNacosInstanceServiceName(instanceServiceName string) NacosInstanceServiceName {
	split := strings.Split(instanceServiceName, "@")
	return NacosInstanceServiceName{
		GroupName:   split[0],
		ServiceName: split[2],
	}
}

// getNacosNamingClient 获取 nacos naming client
func getNacosNamingClient(discovery Discovery) naming_client.INamingClient {
	// nacos client config
	clientConfig := constant.NewClientConfig(
		constant.WithNamespaceId(discovery.Namespace),
		constant.WithTimeoutMs(50000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("error"),
	)
	serverConfig := []constant.ServerConfig{
		*constant.NewServerConfig(
			discovery.Ip,
			discovery.Port,
			constant.WithContextPath("/nacos"),
		),
	}
	// nacos naming client
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  clientConfig,
			ServerConfigs: serverConfig,
		},
	)
	checkErr(err)
	return namingClient
}

// generateNginxConf 生成 nginx conf 配置文件
func generateNginxConf(serviceAddressesMap map[string][]ServiceAddress, nginxConfPath string) {
	for serviceName, serviceAddresses := range serviceAddressesMap {
		subscribeServiceInfo := subscribeServicesMap[serviceName]
		log.Printf("serviceName: %s, serviceAddresses: %s, subscribeServiceInfo: %s", serviceName, util.ToJsonString(serviceAddresses), util.ToJsonString(subscribeServiceInfo))

		var nginxConfig NginxConfig
		nginxConfig.ServiceAddresses = serviceAddresses
		nginxConfig.ServiceName = serviceName
		nginxConfig.NginxServerName = subscribeServiceInfo.NginxServerName
		nginxConfig.NginxPort = subscribeServiceInfo.NginxPort
		nginxConfig.NginxUpstreamName = subscribeServiceInfo.NginxUpstreamName
		serviceNginxConf := nginxConfPath + fmt.Sprintf("/%s.conf", subscribeServiceInfo.NginxUpstreamName)

		file, err := os.Create(serviceNginxConf)
		checkErr(err)
		err = file.Chmod(os.ModePerm)
		checkErr(err)
		tpl := template.Must(template.ParseGlob("./tpl/*.tpl"))
		err = tpl.ExecuteTemplate(file,
			"tpl/nginx_conf",
			map[string]interface{}{
				"NginxConfig": nginxConfig,
			},
		)
		checkErr(err)
	}
}

// getSubscribeServicesMap 获取 SubscribeService
func getSubscribeServicesMap(subscribeServices []SubscribeService) map[string]SubscribeService {
	subscribeServicesMap := map[string]SubscribeService{}
	for _, subscribeService := range subscribeServices {
		subscribeServicesMap[subscribeService.ServiceName] = subscribeService
	}
	return subscribeServicesMap
}

// sendMsgToWeWork 发送通知到企业微信
func sendMsgToWeWork(content string) {
	if !config.WeWork.Enabled {
		return
	}
	now := time.Now()
	format := now.Format("2006-01-02 15:04:05")
	body := fmt.Sprintf(`{"msgtype": "markdown", "markdown": {"content": "### Nginx 运行消息通知 \n<font color=\"comment\">%s</font>\n%s"}}`, format, content)
	res, err := http.Post(config.WeWork.Url, "application/json", bytes.NewBuffer([]byte(body)))
	if err != nil {
		log.Println("WeWork message notification failed.")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		checkErr(err)
	}(res.Body)
}

// reloadNginx 重载 nginx
func reloadNginx(serviceNamesStr string) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	// 校验 nginx conf 配置是否正确
	cmd := exec.Command(config.Nginx.NginxBin, "-t")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := fmt.Sprintf(config.WeWork.Messages["nginx-reload-error"], serviceNamesStr, stderr.String())
		log.Println(msg)
		// 发送企业微信通知
		sendMsgToWeWork(msg)
		return
	}

	// 重启 nginx
	cmd = exec.Command(config.Nginx.NginxBin, "-s", "reload")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		msg := fmt.Sprintf(config.WeWork.Messages["nginx-reload-error"], serviceNamesStr, stderr.String())
		log.Printf(msg)
		// 发送企业微信通知
		sendMsgToWeWork(msg)
		return
	}
}

// fileIsExists 文件是否存在
func fileIsExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil || os.IsExist(err) {
		checkErr(err)
		return true
	}
	return false
}

// initConfig 初始化配置
func initConfig(configFile string) *Config {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	checkErr(err)
	var _config *Config
	err = viper.Unmarshal(&_config, func(decoderConfig *mapstructure.DecoderConfig) {
		decoderConfig.TagName = "yaml"
	})
	checkErr(err)
	log.Printf("_config: %s", util.ToJsonString(_config))
	return _config
}

// checkErr 抛出异常
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
