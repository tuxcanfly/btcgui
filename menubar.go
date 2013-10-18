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
	"github.com/conformal/gotk3/gtk"
	"log"
)

var (
	// MenuBar holds pointers to various items in the menu.
	MenuBar = struct {
		Settings struct {
			New     *gtk.MenuItem
			Encrypt *gtk.MenuItem
			Lock    *gtk.MenuItem
			TxFee   *gtk.MenuItem
			Unlock  *gtk.MenuItem
		}
	}{}
)

func createFileMenu() *gtk.MenuItem {
	menu, err := gtk.MenuItemNewWithMnemonic("_File")
	if err != nil {
		log.Fatal(err)
	}

	dropdown, err := gtk.MenuNew()
	if err != nil {
		log.Fatal(err)
	}

	menu.SetSubmenu(dropdown)

	mitem, err := gtk.MenuItemNewWithMnemonic("E_xit")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		gtk.MainQuit()
	})

	dropdown.Append(mitem)

	return menu
}

func createSettingsMenu() *gtk.MenuItem {
	menu, err := gtk.MenuItemNewWithMnemonic("_Settings")
	if err != nil {
		log.Fatal(err)
	}
	dropdown, err := gtk.MenuNew()
	if err != nil {
		log.Fatal(err)
	}
	menu.SetSubmenu(dropdown)

	mitem, err := gtk.MenuItemNewWithLabel("New Wallet...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		if dialog, err := createNewWalletDialog(); err != nil {
			log.Print(err)
		} else {
			dialog.Run()
		}
	})
	dropdown.Append(mitem)
	mitem.SetSensitive(false)
	MenuBar.Settings.New = mitem

	mitem, err = gtk.MenuItemNewWithLabel("Encrypt Wallet...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		if dialog, err := createEncryptionDialog(); err != nil {
			log.Print(err)
		} else {
			dialog.Run()
		}
	})
	dropdown.Append(mitem)
	mitem.SetSensitive(false)
	MenuBar.Settings.Encrypt = mitem

	mitem, err = gtk.MenuItemNewWithLabel("Lock wallet")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		go func() {
			triggers.lockWallet <- 1
		}()
	})
	dropdown.Append(mitem)
	mitem.SetSensitive(false)
	MenuBar.Settings.Lock = mitem

	mitem, err = gtk.MenuItemNewWithLabel("Unlock Wallet...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		if dialog, err := createUnlockDialog(); err != nil {
			log.Print(err)
		} else {
			dialog.Run()
		}
	})
	dropdown.Append(mitem)
	mitem.SetSensitive(false)
	MenuBar.Settings.Unlock = mitem

	sep, err := gtk.SeparatorMenuItemNew()
	if err != nil {
		log.Fatal(err)
	}
	dropdown.Append(sep)

	mitem, err = gtk.MenuItemNewWithLabel("Set Transaction Fee...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		if dialog, err := createTxFeeDialog(); err != nil {
			log.Print(err)
		} else {
			dialog.Run()
		}
	})
	dropdown.Append(mitem)
	//mitem.SetSensitive(false)
	MenuBar.Settings.TxFee = mitem

	return menu
}

func createHelpMenu() *gtk.MenuItem {
	menu, err := gtk.MenuItemNewWithMnemonic("_Help")
	if err != nil {
		log.Fatal(err)
	}
	dropdown, err := gtk.MenuNew()
	if err != nil {
		log.Fatal(err)
	}
	menu.SetSubmenu(dropdown)

	mitem, err := gtk.MenuItemNewWithLabel("Tutorial...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		w, err := CreateTutorialDialog(mainWindow)
		if err != nil {
			// TODO(jrick): Log error to file.
			log.Fatal(err)
		}
		w.ShowAll()
	})
	dropdown.Append(mitem)

	return menu
}

func createMenuBar() *gtk.MenuBar {
	m, err := gtk.MenuBarNew()
	if err != nil {
		log.Fatal(err)
	}

	m.Append(createFileMenu())
	m.Append(createSettingsMenu())
	m.Append(createHelpMenu())

	return m
}
