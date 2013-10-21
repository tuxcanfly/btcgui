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
)

// NewWalletParams holds the parameters needed to create a new wallet.
type NewWalletParams struct {
	name       string
	desc       string
	passphrase string
}

func createNewWalletDialog() (*gtk.Dialog, error) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	dialog.SetTitle("New wallet")

	dialog.AddButton("_OK", gtk.RESPONSE_OK)

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

	l, err := gtk.LabelNew("Passphrase")
	if err != nil {
		return nil, err
	}
	grid.Attach(l, 0, 0, 1, 1)

	passphrase, err := gtk.EntryNew()
	if err != nil {
		return nil, err
	}
	passphrase.SetVisibility(false)
	passphrase.SetHExpand(true)
	passphrase.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(passphrase, 1, 0, 1, 1)

	l, err = gtk.LabelNew("Repeat passphrase")
	if err != nil {
		return nil, err
	}
	l.SetVExpand(true)
	l.SetVAlign(gtk.ALIGN_START)
	grid.Attach(l, 0, 1, 1, 1)

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
	grid.Attach(repeated, 1, 1, 1, 1)

	showEntryText, err := gtk.CheckButtonNewWithLabel("Show passphrases")
	if err != nil {
		return nil, err
	}
	showEntryText.Connect("toggled", func() {
		active := showEntryText.GetActive()
		passphrase.SetVisibility(active)
		repeated.SetVisibility(active)
	})
	grid.Attach(showEntryText, 0, 2, 2, 1)

	dialog.SetTransientFor(mainWindow)
	dialog.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	dialog.ShowAll()

	dialog.Connect("response", func(ctx *glib.CallbackContext) {
		switch gtk.ResponseType(ctx.Arg(0).Int()) {
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
				go func() {
					triggers.newWallet <- &NewWalletParams{
						name:       "",
						desc:       "",
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
