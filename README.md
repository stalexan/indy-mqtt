# IndyMqtt

IndyMqtt is a command-line utility that can be used to monitor and configure
an [IndySwitch](https://github.com/stalexan/indy-switch), to:

* Report status.
* Turn the switch on and off.
* Configure time settings.
* Reset the switch it its original settings.
* Restart the switch.

[MQTT](https://en.wikipedia.org/wiki/MQTT) is used for this. IndyMqtt and
IndySwitch connect to an an MQTT broker, and the broker relays messages to
and from the IndySwitches.


## Usage

```
NAME
    indy-mqtt - Send commands to an MQTT broker to administer an IndySwitch.

SYNOPSIS
    indy-mqtt [options] [host] [command]

DESCRIPTION
    Monitor and maintain an IndySwitch by sending commands to an MQTT broker.

OPTIONS
    -help
        Show help and exit

    -version
        Print version information and exit

    -verbose
        Print all status messages

    -debug
        Print debug messages

COMMANDS
    config timezone [timezone]
        Sets the timezone.

    config offset [offset]
        Sets the the random offset used to turn the switch on and off.

    config suntimes [filename]
        Configures the sunrise and sunset times.

    status [all] 
        Returns a status report.

    restart
        Restarts the switch.

    reset
        Resets the switch to its original settings, and restarts it.

    switch [on|off]
        Turns the switch on and off.

FILES
    internal/config/config.json
        Configures the hostname and port of the MQTT broker to talk to. For example:
            {
                "hostname": "bettyboop123.com",
                "port": 8883
            }

    internal/config/config-secrets.json
        Configures the credentials used to connect to the MQTT broker. For example:
            {
                "username": "foobar",
                "password": "changeme"
            }
```

## Examples

Turn a switch on:

```
$ indy-mqtt foobar on
```

Turn a switch off:

```
$ indy-mqtt foobar off
```

Display switch status:

```
$ indy-mqtt foobar status
date: Wed Jan 17 10:57:55 2024 CST
is_on: false
sunrise: Thu Jan 18 06:53:00 2024 CST
sunset: Wed Jan 17 18:03:00 2024 CST
offset: 60
next_action: ON
next_action_time: Wed Jan 17 18:44:00 2024 CST
```

Set timezone:

```
$ indy-mqtt foobar config timezone CST6
```

Set random offset to +/- 1 hour:

```
$ indy-mqtt foobar config offset 60
```

Set sunrise and sunset times:

```
$ indy-mqtt foobar configure suntimes suntimes.json
```

Where `suntimes.json` has:

```
{
  "1":  ["6:53 AM", "6:03 PM"],
  "2":  ["6:46 AM", "6:20 PM"],
  "3":  ["6:26 AM", "6:29 PM"],
  "4":  ["6:02 AM", "6:36 PM"],
  "5":  ["5:45 AM", "6:45 PM"],
  "6":  ["5:43 AM", "6:56 PM"],
  "7":  ["5:51 AM", "6:58 PM"],
  "8":  ["6:01 AM", "6:45 PM"],
  "9":  ["6:07 AM", "6:20 PM"],
  "10": ["6:12 AM", "5:57 PM"],
  "11": ["6:25 AM", "5:42 PM"],
  "12": ["6:42 AM", "5:46 PM"]
}
```

Reset the switch to its original flashed settings:

```
$ indy-mqtt foobar reset
```

## Building

IndyMqtt requires Go version 1.21 to build. See [go.dev](https://go.dev/) to install Go: 
[Download and install](https://go.dev/doc/install).

Then clone the repo:

```
git clone https://github.com/stalexan/indy-mqtt.git
```

And build with:

```
cd indy-mqtt
make build
```

## License

IndyMqtt is licensed under the [MIT License](https://spdx.org/licenses/MIT.html).
You can find the complete text in [LICENSE](LICENSE).
