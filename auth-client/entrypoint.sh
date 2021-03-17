#!/bin/sh

iptables -t nat -A POSTROUTING -s ${OPENVPN_SUBNET} ! -d ${OPENVPN_SUBNET} -j MASQUERADE

mkdir -p /dev/net
if [ ! -c /dev/net/tun ]; then
    mknod /dev/net/tun c 10 200
fi

exec openvpn --config /openvpn/server.conf
