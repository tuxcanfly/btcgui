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
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
)

const encryptMessage = "Enter the new passphrase to the wallet.\n" +
	"Please use a passphrase of " +
	"<b>10 or more random characters,</b> " +
	"or " +
	"<b>eight or more words</b>" +
	"."

func createEncryptionDialog() (*gtk.Dialog, error) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	dialog.SetTitle("Encrypt wallet")

	dialog.AddButton("_OK", gtk.RESPONSE_OK)
	dialog.AddButton("_Cancel", gtk.RESPONSE_CANCEL)

	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetHExpand(true)
	grid.SetVExpand(true)
	b, err := dialog.GetContentArea()
	if err != nil {
		return nil, err
	}
	b.Add(grid)
	b.SetHExpand(true)
	b.SetVExpand(true)

	l, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	l.SetMarkup(encryptMessage)
	l.SetHExpand(true)
	l.SetVExpand(true)
	l.SetHAlign(gtk.ALIGN_START)
	grid.Attach(l, 0, 0, 2, 1)

	l, err = gtk.LabelNew("New passphrase")
	if err != nil {
		return nil, err
	}
	grid.Attach(l, 0, 1, 1, 1)

	passphrase, err := gtk.EntryNew()
	if err != nil {
		return nil, err
	}
	passphrase.SetVisibility(false)
	passphrase.SetHExpand(true)
	passphrase.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(passphrase, 1, 1, 1, 1)

	l, err = gtk.LabelNew("Repeat new passphrase")
	if err != nil {
		return nil, err
	}
	l.SetVExpand(true)
	l.SetVAlign(gtk.ALIGN_START)
	grid.Attach(l, 0, 2, 1, 1)

	repeated, err := gtk.EntryNew()
	if err != nil {
		return nil, err
	}
	repeated.SetVisibility(false)
	repeated.SetVExpand(true)
	repeated.SetVAlign(gtk.ALIGN_START)
	repeated.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(repeated, 1, 2, 1, 1)

	dialog.SetTransientFor(mainWindow)
	dialog.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	dialog.ShowAll()

	dialog.Connect("response", func(_ *glib.Object, rt gtk.ResponseType) {
		switch rt {
		case gtk.RESPONSE_OK:
			pStr, err := passphrase.GetText()
			if err != nil {
				log.Print(err)
				return
			}
			rStr, err := repeated.GetText()
			if err != nil {
				log.Print(err)
				return
			}
			if pStr == rStr {
				// use the passphrase, encrypt wallet...
				dialog.Destroy()
			} else {
				msg := "The supplied passphrases do not match."
				mDialog := gtk.MessageDialogNew(dialog, 0,
					gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, msg)
				mDialog.SetTitle("Wallet encryption failed")
				mDialog.Run()
				mDialog.Destroy()
			}
		case gtk.RESPONSE_CANCEL:
			dialog.Destroy()
		}
	})

	return dialog, nil
}
