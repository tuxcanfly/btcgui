/*
 * Copyright (c) 2013 Conformal Systems LLC <info@conformal.com>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"fmt"
	"github.com/conformal/btcutil"
	"github.com/conformal/btcwire"
	"github.com/conformal/go-flags"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigFilename = "btcgui.conf"
	defaultDataDirname    = "data"
	defaultBtcNet         = btcwire.TestNet3
)

var (
	btcguiHomeDir     = btcutil.AppDataDir("btcgui", false)
	defaultConfigFile = filepath.Join(btcguiHomeDir, defaultConfigFilename)
	defaultDataDir    = filepath.Join(btcguiHomeDir, defaultDataDirname)
)

type config struct {
	ShowVersion bool   `short:"V" long:"version" description:"Display version information and exit"`
	ConfigFile  string `short:"C" long:"configfile" description:"Path to configuration file"`
	Port        string `short:"p" long:"port" description:"port to connect "`
	MainNet     bool   `long:"mainnet" description:"*DISABLED* Use the main Bitcoin network (default testnet3)"`
}

// updateConfigWithActiveParams update the passed config with parameters
// from the active net params if the relevant options in the passed config
// object are the default so options specified by the user on the command line
// are not overridden.
func updateConfigWithActiveParams(cfg *config) {
	if cfg.Port == netParams(defaultBtcNet).port {
		cfg.Port = activeNetParams.port
	}
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// loadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
//      1) Start with a default config with sane settings
//      2) Pre-parse the command line to check for an alternative config file
//      3) Load configuration file overwriting defaults with any specified options
//      4) Parse CLI options and overwrite/add any specified options
//
// The above results in btcgui functioning properly without any config
// settings while still allowing the user to override settings with config files
// and command line options.  Command line options always take precedence.
func loadConfig() (*config, []string, error) {
	// Default config.
	cfg := config{
		ConfigFile: defaultConfigFile,
		Port:       netParams(defaultBtcNet).port,
	}

	// A config file in the current directory takes precedence.
	if fileExists(defaultConfigFilename) {
		cfg.ConfigFile = defaultConfigFile
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.Default)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			preParser.WriteHelp(os.Stderr)
		}
		return nil, nil, err
	}

	// Show the version and exit if the version flag was specified.
	if preCfg.ShowVersion {
		appName := filepath.Base(os.Args[0])
		appName = strings.TrimSuffix(appName, filepath.Ext(appName))
		fmt.Println(appName, "version", version)
		os.Exit(0)
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)
	err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintln(os.Stderr, err)
			parser.WriteHelp(os.Stderr)
			return nil, nil, err
		}
		configFileError = err
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return nil, nil, err
	}

	// Warn about missing config file after the final command line parse
	// succeeds.  This prevents the warning on help messages and invalid
	// options.
	if configFileError != nil {
		log.Printf("[WARN] %v", err)
	}

	// TODO(jrick): Enable mainnet support again when ready.
	cfg.MainNet = false

	// Choose the active network params based on the mainnet net flag.
	if cfg.MainNet {
		activeNetParams = netParams(btcwire.MainNet)
	}
	updateConfigWithActiveParams(&cfg)

	return &cfg, remainingArgs, nil

}
