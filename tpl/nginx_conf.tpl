{{ define "tpl/nginx_conf" }}

upstream {{ .NginxConfig.NginxUpstreamName }} {
    {{ range $index, $serviceAddresses  := .NginxConfig.ServiceAddresses }}server {{ $serviceAddresses.Ip }}:{{ $serviceAddresses.Port }} weight={{ $serviceAddresses.Weight }};
    {{ end }}
}

server {
    listen {{ .NginxConfig.NginxPort }};
    {{ if .NginxConfig.NginxServerName }}
    server_name  {{ .NginxConfig.NginxServerName }};
    {{ end }}
    location / {
        proxy_pass http://{{ .NginxConfig.NginxUpstreamName }};
        proxy_set_header Host $host:$server_port;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}

{{ end }}