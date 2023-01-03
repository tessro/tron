# Tron

Tron is a tiny CLI tool for controlling Lutron Caséta systems.

## Installation

Make sure `$GOPATH/bin` is on your `$PATH`, then run:

```bash
go install github.com/paulrosania/tron
```

Tron expects a `.tronrc` file in your home directory, with the following
settings:

```ini
host=<ip address>
```

You can find your Lutron controller's IP address via your router console.

Before you can run commands, you need to pair `tron` with your controller. To do
this, run `tron pair` and follow the instructions in your terminal.

## Usage

```bash
# Setup
tron pair   # Pair with a controller
tron ping   # Verify that `tron` can communicate with your controller

# Devices
tron device list           # List installed devices

# Servers
tron server list           # List available controllers

# Services
tron service list          # List supported 3rd party services

# Raw querying
tron get <path>            # Send a `ReadRequest`
tron post <path> <json>    # Send a `CreateRequest`
```
