# nginx-reload-from-nacos
监听 nacos 服务更新（服务上线/服务下线）以自动更新 nginx conf 配置文件并 reload nginx

## 特性 Features
- [x] 支持监听多个服务
- [x] 支持指定配置文件
- [x] 支持企业微信通知
- [x] 支持启动多个监听实例
- [ ] 自定义 Nacos 重要配置项

## 运行 Run

```shell
# simple to run
./nginx-reload-from-nacos -c "./config.yaml"
```

```shell
# run in background
nohup ./nginx-reload-from-nacos > ./run.log 2>&1 &
```

## 开发 Develop

- Dev/Compile in golang >= 1.19.3
- Dev on Fedora >= 3.7
- Compile on CentOS >= 7.9.2009

## 反馈 Feedback

[Issues](https://github.com/godcheese/nginx-reload-from-nacos/issues)

## 捐赠 Donation

如果此项目对你有所帮助，不妨请我喝咖啡。
If you find this project useful, you can buy us a cup of coffee.

[Paypal Me](https://www.paypal.me/godcheese)

## 协议 License
[MIT License](https://github.com/godcheese/nginx-reload-from-nacos/blob/main/LICENSE) Copyright (c) 2023 godcheese