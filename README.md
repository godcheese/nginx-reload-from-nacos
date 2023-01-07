# nginx-reload-from-nacos
监听 nacos 服务更新（服务上线/服务下线）以自动更新 nginx conf 配置文件并 reload nginx

### run 运行

```shell
# simple to run
./nginx-reload-from-nacos -c "./config.yaml"
```

```shell
# run in background
nohup ./nginx-reload-from-nacos > ./run.log 2>&1 &
```
