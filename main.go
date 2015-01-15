/*
 * Copyright (c) 2013, 2014 Conformal Systems LLC <info@conformal.com>
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
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/conformal/go-flags"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
)

// cfg holds the default and overridden configuration settings set
// from a config file and command line flags.
var cfg *config

var PreGUIErrorDialog *gtk.MessageDialog

// PreGUIError opens the pre-allocated error dialog for presenting errors
// before the main window GUI has been completely constructed and shown.
// The dialog is updated with the message in e.
func PreGUIError(e error) {
	// Update dialog with the message in e.
	PreGUIErrorDialog.SetMarkup(e.Error())

	// Run and destroy dialog.  os.Exit should be called once the
	// dialog is destroyed.
	PreGUIErrorDialog.Run()
	PreGUIErrorDialog.Destroy()
}

// IdlePreGUIError runs PreGUIError within the context of the GTK main
// event loop.  This function does not return.
func IdlePreGUIError(e error) {
	glib.IdleAdd(func() {
		PreGUIError(e)
	})

	// This function should block.  However, simple adding a closure the
	// main event loop does not block.  Use an empty select to prevent the
	// calling goroutine from continuing.
	select {}
}

func main() {
	gtk.Init(nil)

	// The first thing ever done is to create a GTK error dialog to
	// show any errors to the user.  If any fatal errors occur before
	// the main application window is shown, they will be shown using
	// this dialog.
	PreGUIErrorDialog = gtk.MessageDialogNew(nil, 0, gtk.MESSAGE_ERROR,
		gtk.BUTTONS_OK, "An unknown error occured.")
	PreGUIErrorDialog.SetPosition(gtk.WIN_POS_CENTER)
	PreGUIErrorDialog.Connect("destroy", func() {
		os.Exit(1)
	})

	tcfg, _, err := loadConfig()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			PreGUIError(fmt.Errorf("Cannot open configuration:\n%v", err))
		} else {
			os.Exit(1)
		}
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
			PreGUIError(fmt.Errorf("Cannot create tutorial dialog:\n%v", err))
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
	// Read CA file to verify a btcwallet TLS connection.
	cafile, err := ioutil.ReadFile(cfg.CAFile)
	if err != nil {
		IdlePreGUIError(fmt.Errorf("Cannot open CA file:\n%v", err))
	}

	glib.IdleAdd(func() {
		w, err := CreateWindow()
		if err != nil {
			PreGUIError(fmt.Errorf("Cannot create application window:\n%v", err))
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
			ListenAndUpdate(cafile, replies)
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
					log.Print("Established connection to btcwallet.")
				default:
					// TODO(jrick): present unknown error to user in the
					// GUI somehow.
					log.Printf("Unknown connect error: %v", err)
				}
			}
		}
	}
}
