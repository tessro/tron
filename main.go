package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

const defaultConfigFile = ".tronrc"
const defaultCertDir = ".config/tron/certs"

//go:generate bash get_versions.sh

//go:embed tmp/version.txt
var Version string

//go:embed tmp/go_version.txt
var GoVersion string

//go:embed tmp/commit_hash.txt
var CommitHash string

var verbose = flag.Bool("v", false, "Verbose")

func usage() {
	fmt.Println("usage: tron [-v] <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println()
	fmt.Println("   version      Print tron version")
	fmt.Println()
	fmt.Println("   pair         Pair with a Lutron Caséta controller")
	fmt.Println("   ping         Ping paired controller")
	fmt.Println()
	fmt.Println("   get          Query controller endpoints")
	fmt.Println("   post         Send data to controller endpoints")
	fmt.Println()
	fmt.Println("   area         Control areas")
	fmt.Println("   device       Control Lutron devices")
	fmt.Println("   server       Control Lutron controllers")
	fmt.Println("   service      Control 3rd-party services")
	fmt.Println("   zone         Control zones")
	fmt.Println()
	os.Exit(1)
}

func main() {
	flag.Parse()

	usr, err := user.Current()
	if err != nil {
		fmt.Println("error: failed to fetch current user:", err)
		os.Exit(1)
	}
	dir := usr.HomeDir
	configFilePath := filepath.Join(dir, defaultConfigFile)

	cfg, err := ini.Load(configFilePath)
	if err != nil {
		fmt.Println("error: failed to read file:", err)
		os.Exit(1)
	}

	client := Client{
		Host: cfg.Section("").Key("host").String(),

		CACertPath:     filepath.Join(dir, defaultCertDir, "ca.crt"),
		ClientCertPath: filepath.Join(dir, defaultCertDir, "client.crt"),
		ClientKeyPath:  filepath.Join(dir, defaultCertDir, "client.key"),

		Verbose: *verbose,
	}

	if *verbose {
		os.Stderr.WriteString(fmt.Sprintf("Host: %s\n\n", client.Host))
	}

	if flag.NArg() > 0 {
		cmd := flag.Arg(0)
		switch cmd {
		case "pair":
			err := client.Pair()
			if err != nil {
				fmt.Println("error: failed to pair controller:", err)
				os.Exit(1)
			}
		case "area":
			doAreaCommand(client, flag.Args()[1:])
		case "device":
			doDeviceCommand(client, flag.Args()[1:])
		case "server":
			doServerCommand(client, flag.Args()[1:])
		case "service":
			doServiceCommand(client, flag.Args()[1:])
		case "zone":
			doZoneCommand(client, flag.Args()[1:])
		case "get":
			doGetCommand(client, flag.Args()[1:])
		case "post":
			doPostCommand(client, flag.Args()[1:])
		case "ping":
			res, err := client.Ping()
			if err != nil {
				fmt.Println("error: failed to ping controller:", err)
				os.Exit(1)
			}
			fmt.Printf("OK (LEAP version %0.3f)\n", res.LEAPVersion)
		case "version":
			fmt.Printf("tron %s (%s) go/%s\n", Version, CommitHash, GoVersion)
		default:
			usage()
		}
	} else {
		usage()
	}
}

func doAreaCommand(client Client, args []string) {
	printArea := func(area AreaDefinition) {
		fmt.Println("Name:    ", area.Name)
		fmt.Println("Category:", area.Category.Type)
		fmt.Println("Path:    ", area.Href)
		fmt.Println("Parent:  ", area.Parent.Href)
		fmt.Println()
		fmt.Println("Devices:")
		for _, d := range area.AssociatedDevices {
			fmt.Println("-", d.Href)
		}
		fmt.Println()
		fmt.Println("Daylighting Gain Settings:", area.DaylightingGainSettings.Href)
		fmt.Println("Load Shedding:            ", area.LoadShedding.Href)
		fmt.Println("Occupancy Settings:       ", area.OccupancySettings.Href)
		fmt.Println("Occupancy Sensor Settings:", area.OccupancySensorSettings.Href)
		fmt.Println()
		fmt.Println("Occupancy Groups:")
		for _, og := range area.AssociatedOccupancyGroups {
			fmt.Println("-", og.Href)
		}
	}

	usage := func() {
		fmt.Println("usage: tron area list")
		fmt.Println("       tron area info <id>")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "info":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		area, err := client.Area(id)
		if err != nil {
			fmt.Println("error: failed to retrieve area info:", err)
			os.Exit(1)
		}
		printArea(area)
	case "list":
		list, err := client.Areas()
		if err != nil {
			fmt.Println("error: failed retrieve area list:", err)
			os.Exit(1)
		}
		first := true
		for _, area := range list {
			if first {
				first = false
			} else {
				fmt.Println("==========")
				fmt.Println()
			}
			printArea(area)
			fmt.Println()
		}
	default:
		usage()
	}
}

func doDeviceCommand(client Client, args []string) {
	printDevice := func(device DeviceDefinition) {
		fmt.Println("Name:         ", strings.Join(device.FullyQualifiedName, " "))
		fmt.Println("Path:         ", device.Href)
		fmt.Println("Type:         ", device.DeviceType)
		fmt.Println("Model Number: ", device.ModelNumber)
		fmt.Println("Serial Number:", device.SerialNumber)
		fmt.Println()
		fmt.Println("Addressed State:", device.AddressedState)
		fmt.Println("Associated Area:", device.AssociatedArea.Href)
		fmt.Println("Parent Path:    ", device.Parent.Href)
		fmt.Println()
		if len(device.LocalZones) > 0 {
			fmt.Println("Local Zones:")
			for _, lz := range device.LocalZones {
				fmt.Println("-", lz.Href)
			}
		}
		fmt.Println()
		if len(device.ButtonGroups) > 0 {
			fmt.Println("Button Groups:")
			for _, bg := range device.ButtonGroups {
				fmt.Println("-", bg.Href)
			}
		}
		fmt.Println()
		if len(device.DeviceRules) > 0 {
			fmt.Println("Device Rules:")
			for _, dr := range device.DeviceRules {
				fmt.Println("-", dr.Href)
			}
		}
		fmt.Println()
		if len(device.LinkNodes) > 0 {
			fmt.Println("Link Nodes:")
			for _, ln := range device.LinkNodes {
				fmt.Println("-", ln.Href)
			}
		}
	}

	usage := func() {
		fmt.Println("usage: tron device list")
		fmt.Println("       tron device info <id>")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "info":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		device, err := client.Device(id)
		if err != nil {
			fmt.Println("error: failed to retrieve device info:", err)
			os.Exit(1)
		}
		printDevice(device)
	case "list":
		list, err := client.Devices()
		if err != nil {
			fmt.Println("error: failed retrieve device list:", err)
			os.Exit(1)
		}
		first := true
		for _, device := range list {
			if first {
				first = false
			} else {
				fmt.Println("==========")
				fmt.Println()
			}
			printDevice(device)
			fmt.Println()
		}
	default:
		usage()
	}
}

func doServerCommand(client Client, args []string) {
	printServer := func(server ServerDefinition) {
		fmt.Println("Path:   ", server.Href)
		fmt.Println("Type:   ", server.Type)
		fmt.Printf("Enabled: %v\n", server.EnableState == "Enabled")
		fmt.Println()
		fmt.Println("Protocol Version:", server.ProtocolVersion)
		fmt.Println()
		fmt.Println("LEAP:")
		fmt.Println("  Pairing List:", server.LEAPProperties.PairingList.Href)
		fmt.Println()
		fmt.Println("Endpoints:")
		for _, ep := range server.Endpoints {
			fmt.Printf("- %d (%s)\n", ep.Port, ep.Protocol)
		}
		fmt.Println()
		fmt.Println("Network Interfaces:")
		for _, iface := range server.NetworkInterfaces {
			fmt.Println("-", iface.Href)
		}
	}

	usage := func() {
		fmt.Println("usage: tron server list")
		fmt.Println("usage: tron server info [id]")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "info":
		id := "1"
		if len(args) >= 2 {
			id = args[1]
		}
		server, err := client.Server(id)
		if err != nil {
			fmt.Println("error: failed to retrieve server info:", err)
			os.Exit(1)
		}
		printServer(server)
	case "list":
		list, err := client.Servers()
		if err != nil {
			fmt.Println("error: failed to retrieve server list:", err)
			os.Exit(1)
		}
		for _, server := range list {
			printServer(server)
		}
	default:
		usage()
	}
}

func doServiceCommand(client Client, args []string) {
	usage := func() {
		fmt.Println("usage: tron service list")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "list":
		list, err := client.Services()
		if err != nil {
			fmt.Println("error: failed retrieve service list:", err)
			os.Exit(1)
		}
		for _, service := range list {
			fmt.Printf("%s (%s)\n", service.Type, service.Href)
		}
	default:
		usage()
	}
}

func doZoneCommand(client Client, args []string) {
	printZone := func(zone ZoneDefinition) {
		fmt.Println("Name:", zone.Name)
		fmt.Println("Path:", zone.Href)
		fmt.Println("Type:", zone.ControlType)
		if zone.Category.Type != "" {
			fmt.Println("Category:")
			fmt.Println("  Type:    ", zone.Category.Type)
			fmt.Println("  Is Light:", zone.Category.IsLight)
		}
		fmt.Println("Device Path:", zone.Device.Href)
	}

	usage := func() {
		fmt.Println("usage: tron zone list")
		fmt.Println("usage: tron zone info <id>")
		fmt.Println("usage: tron zone status <id>")
		fmt.Println("usage: tron zone on <id> [duration] [delay]")
		fmt.Println("usage: tron zone off <id> [duration] [delay]")
		fmt.Println("usage: tron zone dim <id> <level> [duration] [delay]")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "dim":
		if len(args) < 3 {
			usage()
		}
		id := args[1]
		level, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("error: invalid level:", err)
			os.Exit(1)
		}
		options := DimOptions{
			Level: level,
		}
		if len(args) >= 4 {
			options.Duration = args[3]
		}
		if len(args) >= 5 {
			options.Delay = args[4]
		}
		_, err = client.ZoneDim(id, options)
		if err != nil {
			fmt.Println("error: failed to dim zone:", err)
			os.Exit(1)
		}
	case "info":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		zone, err := client.Zone(id)
		if err != nil {
			fmt.Println("error: failed to retrieve zone info:", err)
			os.Exit(1)
		}
		printZone(zone)
	case "list":
		list, err := client.Zones()
		if err != nil {
			fmt.Println("error: failed retrieve zone list:", err)
			os.Exit(1)
		}
		for _, zone := range list {
			printZone(zone)
			fmt.Println()
		}
	case "on":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		options := DimOptions{
			Level: 100,
		}
		if len(args) >= 3 {
			options.Duration = args[2]
		}
		if len(args) >= 4 {
			options.Delay = args[3]
		}
		_, err := client.ZoneDim(id, options)
		if err != nil {
			fmt.Println("error: failed to dim zone:", err)
			os.Exit(1)
		}
	case "off":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		options := DimOptions{
			Level: 0,
		}
		if len(args) >= 3 {
			options.Duration = args[2]
		}
		if len(args) >= 4 {
			options.Delay = args[3]
		}
		_, err := client.ZoneDim(id, options)
		if err != nil {
			fmt.Println("error: failed to dim zone:", err)
			os.Exit(1)
		}
	case "status":
		if len(args) < 2 {
			usage()
		}
		id := args[1]
		zoneStatus, err := client.ZoneStatus(id)
		if err != nil {
			fmt.Println("error: failed to retrieve zone status:", err)
			os.Exit(1)
		}
		fmt.Println("Level:   ", zoneStatus.Level)
		fmt.Println("Accuracy:", zoneStatus.StatusAccuracy)
		fmt.Println()
		fmt.Println("Status Path:", zoneStatus.Href)
		fmt.Println("Zone Path:  ", zoneStatus.Zone.Href)
	default:
		usage()
	}
}

func doGetCommand(client Client, args []string) {
	usage := func() {
		fmt.Println("usage: tron get <path>")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	path := args[0]
	res, err := client.Get(path)
	if err != nil {
		fmt.Println("error: request failed:", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Println("error: failed to format response as JSON:", err)
		os.Exit(1)
	}

	fmt.Println(string(out))
}

func doPostCommand(client Client, args []string) {
	usage := func() {
		fmt.Println("usage: tron post <path> <json>")
		os.Exit(1)
	}

	if len(args) < 2 {
		usage()
	}

	path := args[0]
	raw := args[1]
	var o map[string]any
	err := json.Unmarshal([]byte(raw), &o)
	if err != nil {
		fmt.Println("error: failed to parse input as JSON:", err)
		os.Exit(1)
	}
	res, err := client.Post(path, o)
	if err != nil {
		fmt.Println("error: request failed:", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Println("error: failed to format response as JSON:", err)
		os.Exit(1)
	}

	fmt.Println(string(out))
}
