package main

type Config struct {
	WeWork WeWork `yaml:"we-work"`
	Nginx  Nginx  `yaml:"nginx"`
	Nacos  Nacos  `yaml:"nacos"`
}

type WeWork struct {
	Enabled  bool              `yaml:"enabled"`
	Url      string            `yaml:"url"`
	Messages map[string]string `yaml:"messages"`
}

type Nginx struct {
	NginxBin string `yaml:"nginx-bin"`
	ConfPath string `yaml:"conf-path"`
}

type Nacos struct {
	Discovery Discovery `yaml:"discovery"`
}

type Discovery struct {
	Ip                string             `yaml:"ip"`
	Port              uint64             `yaml:"port"`
	GroupName         string             `yaml:"group-name"`
	Namespace         string             `yaml:"namespace"`
	SubscribeServices []SubscribeService `yaml:"subscribe-services"`
}

type SubscribeService struct {
	ServiceName       string `yaml:"service-name"`
	NginxServerName   string `yaml:"nginx-server-name"`
	NginxPort         uint64 `yaml:"nginx-port"`
	NginxUpstreamName string `yaml:"nginx-upstream-name"`
}
