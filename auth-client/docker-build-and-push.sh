#!/usr/bin/env bash
tag=ixdx/openvpn-oidc:latest
image_id="$(docker build . | grep ^Successfully | awk '{print $3}')"
docker tag $image_id $tag
docker push $tag
