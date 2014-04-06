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

// NewWalletParams holds the parameters needed to create a new wallet.
type NewWalletParams struct {
	passphrase string
}

const newWalletMessage = "Before creating a new wallet, a passphrase " +
	"must be entered.  This passhprase will be used to encrypt " +
	"wallet private keys, and will be required before transactions " +
	"can be created from your wallet.\n" +
	"\n" +
	"A strong passphrase is highly recommended.  Choosing an easily " +
	"guessable passphrase increases the likeliness of a successful " +
	"brute force attack.\n" +
	"\n" +
	"<span weight=\"bold\" fgcolor=\"red\">WARNING:</span> Do not " +
	"lose or forget this passphrase, or your Bitcoins will be " +
	"unspendable."

func createNewWalletDialog() (*gtk.Dialog, error) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	dialog.SetTitle("New wallet")

	dialog.AddButton("_OK", gtk.RESPONSE_OK)

	dialog.SetDefaultGeometry(500, 100)

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

	// Because the label will wrap and the final minimum heights
	// and widths will be absurdly large, first give a size request and
	// show the grid (allocating space for the requested size).  This will
	// make text wrapping labels size nicely inside the grid.
	grid.SetSizeRequest(500, 100)
	grid.Show()

	l, err := gtk.LabelNew("")
	if err != nil {
		return nil, err
	}
	l.SetLineWrap(true)
	l.SetMarkup(newWalletMessage)
	l.SetAlignment(0, 0)
	grid.Attach(l, 0, 0, 2, 1)

	b.SetHExpand(true)
	b.SetVExpand(true)

	l, err = gtk.LabelNew("Enter passphrase:")
	if err != nil {
		return nil, err
	}
	l.SetAlignment(1.0, 0.5)
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

	l, err = gtk.LabelNew("Confirm passphrase:")
	if err != nil {
		return nil, err
	}
	l.SetAlignment(1.0, 0.5)
	grid.Attach(l, 0, 2, 1, 1)

	repeated, err := gtk.EntryNew()
	if err != nil {
		return nil, err
	}
	repeated.SetVisibility(false)
	repeated.SetVAlign(gtk.ALIGN_START)
	repeated.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(repeated, 1, 2, 1, 1)

	showEntryText, err := gtk.CheckButtonNewWithLabel("Show passphrase")
	if err != nil {
		return nil, err
	}
	showEntryText.Connect("toggled", func() {
		active := showEntryText.GetActive()
		passphrase.SetVisibility(active)
		repeated.SetVisibility(active)
	})
	grid.Attach(showEntryText, 1, 3, 2, 1)

	dialog.SetTransientFor(mainWindow)
	dialog.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	dialog.ShowAll()

	// Use an IObject as the receiver object.  This may be called with both
	// a *glib.Object and *gtk.Dialog due to where the signals originate
	// from.
	dialog.Connect("response", func(_ glib.IObject, rt gtk.ResponseType) {
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
			if len(pStr) == 0 {
				mDialog := gtk.MessageDialogNew(dialog, 0,
					gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
					"A passphrase must be entered to create a wallet.")
				mDialog.SetTitle("Wallet creation failed")
				mDialog.Run()
				mDialog.Destroy()
				return
			}
			if pStr == rStr {
				go func() {
					triggers.newWallet <- &NewWalletParams{
						passphrase: pStr,
					}

					if err := <-triggerReplies.walletCreationErr; err != nil {
						glib.IdleAdd(func() {
							mDialog := gtk.MessageDialogNew(dialog, 0,
								gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
								err.Error())
							mDialog.SetTitle("Wallet creation failed")
							mDialog.Run()
							mDialog.Destroy()
						})
					} else {
						glib.IdleAdd(func() {
							dialog.Destroy()
						})
					}
				}()
			} else {
				msg := "The supplied passphrases do not match."
				mDialog := gtk.MessageDialogNew(dialog, 0,
					gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, msg)
				mDialog.SetTitle("Wallet creation failed")
				mDialog.Run()
				mDialog.Destroy()
			}
		case gtk.RESPONSE_CANCEL:
			dialog.Destroy()
		}
	})

	dialog.Connect("delete-event", func() {
		mDialog := gtk.MessageDialogNew(mainWindow, 0,
			gtk.MESSAGE_INFO, gtk.BUTTONS_OK,
			"btcgui cannot be used without a wallet and will now close.")
		mDialog.Show()
		mDialog.Run()
		mDialog.Destroy()
		gtk.MainQuit()
	})

	return dialog, nil
}
