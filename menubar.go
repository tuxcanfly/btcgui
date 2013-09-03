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
	MenuBar = struct {
		Settings struct {
			New     *gtk.MenuItem
			Encrypt *gtk.MenuItem
			Lock    *gtk.MenuItem
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
		dialog := createNewWalletDialog()
		dialog.Run()
	})
	dropdown.Append(mitem)
	MenuBar.Settings.New = mitem

	mitem, err = gtk.MenuItemNewWithLabel("Encrypt Wallet...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		dialog := createEncryptionDialog()
		dialog.Run()
	})
	dropdown.Append(mitem)
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
	MenuBar.Settings.Lock = mitem

	mitem, err = gtk.MenuItemNewWithLabel("Unlock Wallet...")
	if err != nil {
		log.Fatal(err)
	}
	mitem.Connect("activate", func() {
		dialog := createUnlockDialog()
		dialog.Run()
	})
	dropdown.Append(mitem)
	MenuBar.Settings.Unlock = mitem

	return menu
}

func createHelpMenu() *gtk.MenuItem {
	mitem, err := gtk.MenuItemNewWithMnemonic("_Help")
	if err != nil {
		log.Fatal(err)
	}

	dropdown, err := gtk.MenuNew()
	if err != nil {
		log.Fatal(err)
	}

	mitem.SetSubmenu(dropdown)

	return mitem
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
