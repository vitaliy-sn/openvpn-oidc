# OpenVPN OpenID Connect (OIDC) Auth

Tested with `google` and [Dex](https://github.com/dexidp/dex/).

### Consists of two components:
* `auth-server` — implements authentication by OpenID Connect (OIDC) provider and outputs information for connecting to the OpenVPN server (username, one-time password and OpenVPN client configuration file).
* `auth-client` — checks the username and password received from OpenVPN client.

### Environment variables are used to configure auth-server:
* `ISSUER_URL` — URL where auth-server can find the OpenID Provider Configuration Document, which should be available in the /.well-known/openid-configuration.
* `DOMAIN` — auth-server domain.
* `CLIENT_ID` — unique identifier for your registered application.
* `CLIENT_SECRET` — is a secret known only to the application and the authentication server.
* `ADDITIONAL_SCOPES` — list of additional scopes.
* `OPENVPN_SERVER_HOST` — IP or domain for connect to OpenVPN server.
* `OPENVPN_SERVER_PORT` — port that listen OpenVPN server.

#### Example for Google:
```shell
ISSUER_URL="https://accounts.google.com"
DOMAIN=openvpn-auth.example.com
CLIENT_ID="0-r.apps.googleusercontent.com"
CLIENT_SECRET="secret"
ADDITIONAL_SCOPES="email"
EXTERNAL_HOST=openvpn.example.com
EXTERNAL_PORT=1194
```

#### Easy install in kubernetes from Helm chart:
```shell
git clone git@github.com:vitaliy-sn/openvpn-oidc.git
cd helm
vim values.yaml
helm install openvpn-oidc .
```

#### How to register app in OIDC provider:
* [Google](https://developers.google.com/identity/protocols/oauth2/openid-connect).
* Dex - you need to create a custom resource oauth2clients.dex.coreos.com in kubernetes cluster.
