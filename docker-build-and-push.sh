#!/usr/bin/env bash

image_id="$(docker build . | grep ^Successfully | awk '{print $3}')"
docker tag $image_id ixdx/openvpn-oidc:latest
docker push ixdx/openvpn-oidc:latest
