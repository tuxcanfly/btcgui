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

// UnlockParams holds parameters necessary to unlock a wallet.
type UnlockParams struct {
	passphrase string
	timeout    int64
}

// UnlockText specifies the title and message to be shown in an
// unlock wallet dialog.
type UnlockText struct {
	Title   string
	Message string
}

var (
	unlockManual = &UnlockText{
		Title: "Unlock wallet",
		Message: "Enter the wallet passphrase and a timeout in seconds.\n" +
			"The wallet will automatically lock after the timeout has expired.",
	}
	unlockForTxSend  = unlockManual
	unlockForKeypool = &UnlockText{
		Title: "Refill address key pool",
		Message: "Wallet must be unlocked to generate new addresses.\n" +
			"The wallet will automatically lock after the timeout has expired.",
	}
)

// createUnlockDialog creates a dialog to enter a passphrase and unlock
// an encrypted wallet.  If an OK response is received, the passphrase will
// be used to attempt a wallet unlock.
//
// If success is non-nil, the caller may pass in a channel to receive a
// notification for whether the unlock was successful.  If the dialog is
// closed without sending a request to btcwallet and the channel is
// non-nil, the channel is closed.
func createUnlockDialog(reason *UnlockText,
	success chan bool) (*gtk.Dialog, error) {

	dialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	dialog.SetTitle(reason.Title)

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

	lbl, err := gtk.LabelNew(reason.Message)
	if err != nil {
		return nil, err
	}
	grid.Attach(lbl, 0, 0, 2, 1)

	lbl, err = gtk.LabelNew("Passphrase")
	if err != nil {
		return nil, err
	}
	grid.Attach(lbl, 0, 1, 1, 1)

	passphrase, err := gtk.EntryNew()
	if err != nil {
		return nil, err
	}
	passphrase.SetVisibility(false)
	passphrase.SetHExpand(true)
	passphrase.SetVExpand(true)
	passphrase.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(passphrase, 1, 1, 1, 1)

	lbl, err = gtk.LabelNew("Timeout (s)")
	if err != nil {
		return nil, err
	}
	grid.Attach(lbl, 0, 2, 1, 1)

	timeout, err := gtk.SpinButtonNewWithRange(0, float64(1<<64-1), 1)
	if err != nil {
		return nil, err
	}
	timeout.SetValue(60)
	timeout.Connect("activate", func() {
		dialog.Emit("response", gtk.RESPONSE_OK, nil)
	})
	grid.Attach(timeout, 1, 2, 1, 1)

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

			timeoutSecs := timeout.GetValueAsInt()

			go func() {
				triggers.unlockWallet <- &UnlockParams{
					pStr,
					int64(timeoutSecs),
				}

				if ok := <-triggerReplies.unlockSuccessful; ok {
					if success != nil {
						success <- true
					}
					glib.IdleAdd(func() {
						dialog.Destroy()
					})
				} else {
					if success != nil {
						success <- false
					}
					glib.IdleAdd(func() {
						mDialog := gtk.MessageDialogNew(dialog, 0,
							gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
							"Wallet decryption failed.")
						mDialog.SetTitle("Wallet decryption failed")
						mDialog.Run()
						mDialog.Destroy()
					})
				}
			}()

		case gtk.RESPONSE_CANCEL:
			if success != nil {
				close(success)
			}
			dialog.Destroy()
		}
	})

	return dialog, nil
}
