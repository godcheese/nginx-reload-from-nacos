package main

type NginxConfig struct {
	ServiceName       string
	NginxServerName   string
	NginxPort         uint64
	NginxUpstreamName string
	ServiceAddresses  []ServiceAddress
}

type ServiceAddress struct {
	Ip     string
	Port   uint64
	Weight uint64
}

type NacosInstanceServiceName struct {
	GroupName   string
	ServiceName string
}
