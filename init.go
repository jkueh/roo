package main

import (
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/jkueh/roo/util"
)

func init() {
	// Set debug mode via environment variable
	debug = strings.ToLower(os.Getenv("DEBUG")) == "true"
	verbose = strings.ToLower(os.Getenv("VERBOSE")) == "true"

	if debug {
		log.Println("Debug mode enabled.")
	}
	if verbose {
		log.Println("Verbose mode enabled.")
	}

	var homeDir string
	var err error

	if homeDir == "" {
		if runtime.GOOS == "windows" {
			homeDir = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if homeDir == "" {
				homeDir = os.Getenv("USERPROFILE")
			}
		} else { // We assume *NIX in bash otherwise
			homeDir = os.Getenv("HOME")
		}
		if homeDir == "" {
			log.Fatalln("Internal error: Unable to determine homeDir")
		}
	}

	if configDir == "" {
		configDir = strings.Join([]string{homeDir, ".roo"}, string(os.PathSeparator))
	}
	err = util.EnsureDirExists(configDir, 0700)
	if err != nil {
		log.Fatalln("Unable to create configDir:", err)
	}
	// Set the default config file path.
	if configFile == "" {
		configFile = strings.Join([]string{configDir, "config.yaml"}, string(os.PathSeparator))
	}

	if cacheDir == "" {
		cacheDir = strings.Join([]string{configDir, "cache"}, string(os.PathSeparator))
	}
	err = util.EnsureDirExists(cacheDir, 0700)
	if err != nil {
		log.Fatalln("Unable to create cacheDir:", err)
	}
}
