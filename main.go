package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"git.yale.edu/spinup/reaper/common"
	"git.yale.edu/spinup/reaper/reaper"
	log "github.com/sirupsen/logrus"
)

// Version is the main version number
const Version = reaper.Version

// VersionPrerelease is a prerelease marker
const VersionPrerelease = reaper.VersionPrerelease

var (
	configFileName = flag.String("config", "config/config.json", "Configuration file.")
	version        = flag.Bool("version", false, "Display version information and exit.")
)

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Reaper version %s%s", Version, VersionPrerelease)

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatalln("Unable to open config file", err)
	}

	r := bufio.NewReader(configFile)
	config, err := common.ReadConfig(r)
	if err != nil {
		log.Fatalf("Unable to read configuration from %s.  %+v", *configFileName, err)
	}

	// Set the loglevel, info if it's unset
	switch config.LogLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.Debugf("Config: %+v", config)
}

func vers() {
	fmt.Printf("Reaper Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}
