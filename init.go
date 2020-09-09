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

	// Some defaults
	if tokenRefreshWindowMinutes == 0 {
		// Identity Account tokens are typically valid for 12 hours.
		tokenRefreshWindowMinutes = 60
	}

	var homeDir string
	var identityAccountCacheFileName string
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
	// Create the config file if it doesn't exist.
	if configFile == "" {
		configFile = strings.Join([]string{configDir, "config.yaml"}, string(os.PathSeparator))
	}
	err = util.EnsureFileExists(configFile, 0600)
	if err != nil {
		log.Fatalln("Unable to create configFile:", err)
	}

	if cacheDir == "" {
		cacheDir = strings.Join([]string{configDir, "cache"}, string(os.PathSeparator))
	}
	err = util.EnsureDirExists(cacheDir, 0700)
	if err != nil {
		log.Fatalln("Unable to create cacheDir:", err)
	}

	// Bootstrap the identityAccountCacheFile
	if identityAccountCacheFileName == "" {
		identityAccountCacheFileName = "identity-account.json"
	}
	identityAccountCacheFile = strings.Join([]string{cacheDir, identityAccountCacheFileName}, string(os.PathSeparator))
	err = util.EnsureFileExists(identityAccountCacheFile, 0600)
	if err != nil {
		log.Fatalln("Unable to create identityAccountCacheFile:", err)
	}
}
