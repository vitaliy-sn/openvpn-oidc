#!/bin/bash

if [ ! -d easyrsa ]; then
  mkdir easyrsa
fi

cd easyrsa

if [ ! -f easyrsa ]; then
  curl -sL https://github.com/OpenVPN/easy-rsa/releases/download/v3.0.6/EasyRSA-unix-v3.0.6.tgz | tar -xzv --strip-components=1 -C .
fi

if [ -d pki ]; then
  exit 0
fi

./easyrsa init-pki
echo -e "ca\n" | ./easyrsa build-ca nopass
./easyrsa build-server-full server nopass
./easyrsa gen-dh

openvpn --genkey --secret ta.key

echo "
  secret:
    ca: |
$(cat pki/ca.crt | sed 's/^/      /')
    cert: |
$(cat pki/issued/server.crt | sed 's/^/      /')
    key: |
$(cat pki/private/server.key | sed 's/^/      /')
    dh: |
$(cat pki/dh.pem | sed 's/^/      /')
    tls: |
$(cat ta.key | sed 's/^/      /')
"
