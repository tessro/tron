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
host=<hostname or ip address>
```

You can find your Lutron controller's IP address via your router console.
Alternatively, you may be able to use mDNS service discovery. For example, on
macOS you can do the following:

```bash
$ dns-sd -Z _lutron | grep -o 'Lutron-.*\.local'
# => Lutron-00000000.local
#
# Use this as your `host` setting.
#
# (You'll need to Ctrl-C to wrap up, since `dns-sd` listens indefinitely.)
```

Before you can run commands, you need to pair `tron` with your controller. To do
this, run `tron pair` and follow the instructions in your terminal.

## Usage

```bash
# Setup
tron pair   # Pair with a controller
tron ping   # Verify that `tron` can communicate with your controller

# Areas
tron area list             # List defined areas
tron area info <id>        # Print information about a specific area

# Devices
tron device list           # List installed devices
tron device info <id>      # Print information about a specific device

# Servers
tron server list           # List available controllers
tron server info [id]      # Print information about a specific controller

# Services
tron service list          # List supported 3rd party services

# Zones
tron zone list             # List defined zones
tron zone info <id>        # Print information about a specific zone
tron zone status <id>      # Print zone status (e.g. dimming level)
tron zone on <id> [duration] [delay]          # Turn the zone on (dim to 100)
tron zone off <id> [duration] [delay]         # Turn the zone off (dim to 0)
tron zone dim <id> <level> [duration] [delay] # Dim the zone to the provided level (0-100)

# Raw querying
tron get <path>            # Send a `ReadRequest`
tron post <path> <json>    # Send a `CreateRequest`
```
