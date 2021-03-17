client
remote {{ .Host }} {{ .Port }}
verb 4
dev tun
dev-type tun
proto tcp
tun-mtu 1500
mssfix
resolv-retry infinite
nobind
persist-key
persist-tun
ns-cert-type server
cipher AES-128-CBC
redirect-gateway def1
keepalive 10 40
key-direction 1
auth-user-pass
<ca>
{{ .CA }}
</ca>
<tls-auth>
{{ .TLSAuth }}
</tls-auth>
