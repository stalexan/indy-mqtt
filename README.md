# indy-mqtt

Command-line client for configuring and controlling an
[indy-switch](https://github.com/stalexan/indy-switch) using MQTT,
written in Go.

# Configuring Secrets

To configure the user name and password connect to the MQTT broker, create
a file called `config-secrets.json` in the directory `internal/config/`, and
add the username and password using this template:
```
{
  "username": "foobar",
  "password": "changeme"
}
```
