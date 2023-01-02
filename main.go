package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/ini.v1"
)

const defaultConfigFile = ".tronrc"
const defaultCertDir = ".config/tron/certs"

var verbose = flag.Bool("v", false, "Verbose")

func usage() {
	fmt.Println("usage: tron [-v] <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println()
	fmt.Println("   pair         Pair with a Lutron Caséta controller")
	fmt.Println("   ping         Ping paired controller")
	fmt.Println()
	fmt.Println("   device       Control Lutron devices")
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
		fmt.Printf("Host: %s\n\n", client.Host)
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
		case "device":
			doDeviceCommand(client, flag.Args()[1:])
		case "ping":
			_, err := client.Ping()
			if err != nil {
				fmt.Println("error: failed to ping controller:", err)
				os.Exit(1)
			}
		default:
			usage()
		}
	} else {
		usage()
	}
}

func doDeviceCommand(client Client, args []string) {
	usage := func() {
		fmt.Println("usage: tron device list")
		os.Exit(1)
	}

	if len(args) < 1 {
		usage()
	}

	command := args[0]
	switch command {
	case "list":
		list, err := client.Devices()
		if err != nil {
			fmt.Println("error: failed retrieve device list:", err)
			os.Exit(1)
		}
		for _, name := range list {
			fmt.Println(name)
		}
	default:
		usage()
	}
}
