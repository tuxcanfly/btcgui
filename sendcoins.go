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
	"container/list"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcutil"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
)

type recipient struct {
	gtk.Widget
	n      int
	payTo  *gtk.Entry
	label  *gtk.Entry
	amount *gtk.SpinButton
	combo  *gtk.ComboBox
}

var (
	recipients = list.New()

	// SendCoins holds pointers to widgets in the send coins tab.
	SendCoins = struct {
		Balance   *gtk.Label
		SendBtn   *gtk.Button
		EntryGrid *gtk.Grid
	}{}
)

func removeRecipentFn(grid *gtk.Grid) func(*gtk.Button, *recipient) {
	return func(_ *gtk.Button, r *recipient) {
		for e := recipients.Front(); e != nil; e = e.Next() {
			if r == e.Value {
				recipients.Remove(e)
				break
			}
		}
		r.Widget.Destroy()

		if recipients.Len() == 0 {
			insertSendEntries(grid)
		}
	}
}

func createRecipient(rmFn func(*gtk.Button, *recipient)) *recipient {
	ret := new(recipient)
	ret.n = recipients.Len()

	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	ret.Widget = grid.Container.Widget

	l, err := gtk.LabelNew("Pay To:")
	if err != nil {
		log.Fatal(err)
	}
	grid.Attach(l, 0, 0, 1, 1)
	l, err = gtk.LabelNew("Amount:")
	if err != nil {
		log.Fatal(err)
	}
	grid.Attach(l, 0, 1, 1, 1)

	payTo, err := gtk.EntryNew()
	if err != nil {
		log.Fatal(err)
	}
	payTo.SetHExpand(true)
	ret.payTo = payTo
	grid.Attach(payTo, 1, 0, 1, 1)

	remove, err := gtk.ButtonNew()
	if err != nil {
		log.Fatal(err)
	}
	img, err := gtk.ImageNewFromIconName("_Delete", gtk.ICON_SIZE_MENU)
	if err != nil {
		log.Fatal(err)
	}
	remove.SetImage(img)
	remove.SetTooltipText("Remove this recipient")
	remove.Connect("clicked", rmFn, ret)
	grid.Attach(remove, 2, 0, 1, 1)

	// TODO(jrick): Label doesn't do anything currently, so don't add
	// to gui.
	/*
		l, err = gtk.LabelNew("Label:")
		if err != nil {
			log.Fatal(err)
		}
		grid.Attach(l, 0, 1, 1, 1)
		label, err := gtk.EntryNew()
		if err != nil {
			log.Fatal(err)
		}
		label.SetHExpand(true)
		ret.label = label
		grid.Attach(label, 1, 1, 2, 1)
	*/

	amounts, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	amount, err := gtk.SpinButtonNewWithRange(0, 21000000, 0.00000001)
	if err != nil {
		log.Fatal(err)
	}
	amount.SetHAlign(gtk.ALIGN_START)
	ret.amount = amount
	amounts.Add(amount)

	ls, err := gtk.ListStoreNew(glib.TYPE_STRING)
	if err != nil {
		log.Fatal(err)
	}
	iter := ls.Append()
	choices := []string{"BTC", "mBTC", "μBTC"}
	s := make([]interface{}, len(choices))
	for i, v := range choices {
		s[i] = v
	}
	if err := ls.Set(iter, []int{0}, []interface{}{"BTC"}); err != nil {
		fmt.Println(err)
	}
	iter = ls.Append()
	if err := ls.Set(iter, []int{0}, []interface{}{"mBTC"}); err != nil {
		fmt.Println(err)
	}
	iter = ls.Append()
	if err := ls.Set(iter, []int{0}, []interface{}{"μBTC"}); err != nil {
		fmt.Println(err)
	}

	// TODO(jrick): add back when this works.
	/*
		combo, err := gtk.ComboBoxNewWithModel(ls)
		if err != nil {
			log.Fatal(err)
		}
		cell, err := gtk.CellRendererTextNew()
		if err != nil {
			log.Fatal(err)
		}
		combo.PackStart(cell, true)
		combo.AddAttribute(cell, "text", 0)
		combo.SetActive(0)
		combo.Connect("changed", func() {
			val := amount.GetValue()
			fmt.Println(val)
			switch combo.GetActive() {
			case 0:
				fmt.Println("btc")
			case 1:
				fmt.Println("mbtc")
			case 2:
				fmt.Println("ubtc")
			}
		})
		ret.combo = combo
		amounts.Add(combo)
	*/
	l, err = gtk.LabelNew("BTC")
	if err != nil {
		log.Fatal(err)
	}
	amounts.Add(l)

	grid.Attach(amounts, 1, 1, 1, 1)

	return ret
}

func insertSendEntries(grid *gtk.Grid) {
	rmFn := removeRecipentFn(grid)
	r := createRecipient(rmFn)

	r.SetHExpand(true)
	r.SetHAlign(gtk.ALIGN_FILL)

	recipients.PushBack(r)

	grid.Add(r)
	grid.ShowAll()
}

func createSendCoins() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	sw, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetHExpand(true)
	sw.SetVExpand(true)
	grid.Add(sw)

	entriesGrid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	SendCoins.EntryGrid = entriesGrid
	entriesGrid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	sw.Add(entriesGrid)
	insertSendEntries(entriesGrid)

	bot, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}

	btn, err := gtk.ButtonNewWithLabel("Add Recipient")
	if err != nil {
		log.Fatal(err)
	}
	btn.SetSizeRequest(150, -1)
	btn.Connect("clicked", func() {
		insertSendEntries(entriesGrid)
	})
	bot.Add(btn)

	l, err := gtk.LabelNew("Balance: ")
	if err != nil {
		log.Fatal(err)
	}
	bot.Add(l)
	SendCoins.Balance = l

	submitBtn, err := gtk.ButtonNewWithLabel("Send")
	if err != nil {
		log.Fatal(err)
	}
	submitBtn.SetSizeRequest(150, -1)
	submitBtn.SetHAlign(gtk.ALIGN_END)
	submitBtn.SetHExpand(true)
	submitBtn.SetSensitive(false)
	submitBtn.Connect("clicked", func() {
		sendTo := make(map[string]float64)
		for e := recipients.Front(); e != nil; e = e.Next() {
			r := e.Value.(*recipient)

			// Get and validate address
			addrStr, err := r.payTo.GetText()
			if err != nil {
				d := errorDialog("Error getting payment address", err.Error())
				d.Run()
				d.Destroy()
				return
			}

			addr, err := btcutil.DecodeAddress(addrStr, activeNet.Params)
			if err != nil {
				d := errorDialog("Invalid payment address",
					fmt.Sprintf("'%v' is not a valid payment address", addrStr))
				d.Run()
				d.Destroy()
				return
			}
			if !addr.IsForNet(activeNet.Params) {
				d := errorDialog("Bad address",
					fmt.Sprintf("Address '%s' is for wrong bitcoin network", addrStr))
				d.Run()
				d.Destroy()
				return
			}

			// Get amount and units and convert to float64
			amt := r.amount.GetValue()
			// Combo box isn't used right now.
			/*
				switch r.combo.GetActive() {
				case 0: // BTC
					// nothing
				case 1: // mBTC
					amt /= 1000
				case 2: // uBTC
					amt /= 1000000
				}
			*/

			sendTo[addrStr] = amt
		}

		go txSenderAndReplyListener(sendTo)
	})
	SendCoins.SendBtn = submitBtn
	bot.Add(submitBtn)

	grid.Add(bot)

	return &grid.Container.Widget
}

// txSenderAndReplyListener triggers btcgui to send btcwallet a JSON
// request to create and send a transaction.  If sending the transaction
// succeeds, the recipients in the send coins notebook tab are cleared.
// If the transaction fails because the wallet is not unlocked, the
// unlock dialog is shown, and after a successful unlock, creating and
// sending the tx is tried a second time.
//
// This is written to be run as a goroutine executing outside of the GTK
// main event loop.
func txSenderAndReplyListener(sendTo map[string]float64) {
	triggers.sendTx <- sendTo

	err := <-triggerReplies.sendTx
	// -13 is the error code for needing an unlocked wallet.
	if jsonErr, ok := err.(*btcjson.Error); ok {
		switch jsonErr.Code {
		case -13:
			// Wallet must be unlocked first.  Show unlock dialog.
			glib.IdleAdd(func() {
				unlockSuccessful := make(chan bool)
				go func() {
					for {
						success, ok := <-unlockSuccessful
						if !ok {
							// A closed channel indicates
							// the dialog was cancelled.
							// Abort sending the transaction.
							return
						}
						if success {
							// Try send again.
							go txSenderAndReplyListener(sendTo)
							return
						}
					}
				}()
				d, err := createUnlockDialog(unlockForTxSend, unlockSuccessful)
				if err != nil {
					// TODO(jrick): log error to file
					log.Printf("[ERR] could not create unlock dialog: %v\n", err)
					return
				}
				d.Run()
				d.Destroy()
			})

		default:
			// Generic case to display an error.
			glib.IdleAdd(func() {
				d := errorDialog("Unable to send transaction",
					fmt.Sprintf("%s\nError code: %d", jsonErr.Message, jsonErr.Code))
				d.Run()
				d.Destroy()
			})
		}
		return
	}

	// Send was successful, so clear recipient widgets.
	glib.IdleAdd(resetRecipients)
}

// resetRecipients resets the recipients list and widgets in the send
// coins tab to a single empty recipient.
//
// This must be run from the GTK main event loop.
func resetRecipients() {
	for e := recipients.Front(); e != nil; e = e.Next() {
		r := e.Value.(*recipient)
		r.Widget.Destroy()
	}
	recipients.Init()
	insertSendEntries(SendCoins.EntryGrid)
}

func errorDialog(title, msg string) *gtk.MessageDialog {
	mDialog := gtk.MessageDialogNew(mainWindow, 0,
		gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
		msg)
	mDialog.SetTitle(title)
	return mDialog
}
