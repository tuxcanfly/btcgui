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
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
	"time"
)

// cfg holds the default and overridden configuration settings set
// from a config file and command line flags.
var cfg *config

func main() {
	gtk.Init(nil)

	tcfg, _, err := loadConfig()
	if err != nil {
		// TODO(jrick): If config fails to load, open warning dialog
		// window instead of dying and never showing anything.
		log.Fatal(err)
	}
	cfg = tcfg

	// Load help dialog on first open.  Use current and previous versions
	// can be used to control what level of new information must be
	// displayed.
	//
	//
	// As currently implemented, if current > previous version, or if
	// there are any errors opening and reading the file, any and all
	// tutorial information is displayed.
	prevRunVers, err := GetPreviousAppVersion(cfg)
	if err != nil || version.NewerThan(*prevRunVers) {
		d, err := CreateTutorialDialog(nil)
		if err != nil {
			// Nothing to show.
			// TODO(jrick): Log warning to file.
			log.Fatal(err)
		}
		d.ShowAll()
		d.Run()
	} else {
		// No error or tutorial dialogs required, so create and show
		// main application window.
		go StartMainApplication()
	}

	gtk.Main()
}

// StartMainApplication creates and opens the main window appWindow.
// It then preceeds to start all necessary goroutine to support the main
// application.  Currently, this starts generating the JSON ID generator
// and attempts to open a connection to btcwallet.
//
// This is written to be called as a goroutine outside of the main GTK
// loop.
func StartMainApplication() {
	glib.IdleAdd(func() {
		w, err := CreateWindow()
		if err != nil {
			// TODO(jrick): log error to file.
			log.Fatal(err)
		}
		w.ShowAll()
	})

	// Write current application version to file.
	if err := version.SaveToDataDir(cfg); err != nil {
		log.Print(err)
	}

	// Begin generating new IDs for JSON calls.
	go JSONIDGenerator(NewJSONID)

	// Listen for updates and update GUI with new info.  Attempt
	// reconnect if connection is lost or cannot be established.
	for {
		replies := make(chan error)
		done := make(chan int)
		go func() {
			ListenAndUpdate(replies)
			close(done)
		}()
	selectLoop:
		for {
			select {
			case <-done:
				break selectLoop
			case err := <-replies:
				switch err {
				case ErrConnectionRefused:
					updateChans.btcwalletConnected <- false
					time.Sleep(5 * time.Second)
				case ErrConnectionLost:
					updateChans.btcwalletConnected <- false
					time.Sleep(5 * time.Second)
				case nil:
					// connected
					updateChans.btcwalletConnected <- true
				default:
					// TODO(jrick): present unknown error to user in the
					// GUI somehow.
					log.Printf("Unknown connect error: %v", err)
				}
			}
		}
	}
}
