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
	"container/list"
	"fmt"
	"github.com/conformal/btcutil"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
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
		Balance *gtk.Label
		SendBtn *gtk.Button
	}{}
)

func removeRecipentFn(grid *gtk.Grid) func(*glib.CallbackContext) {
	return func(ctx *glib.CallbackContext) {
		r := ctx.Data().(*recipient)
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

func createRecipient(rmFn func(*glib.CallbackContext)) *recipient {
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
	l, err = gtk.LabelNew("Label:")
	if err != nil {
		log.Fatal(err)
	}
	grid.Attach(l, 0, 1, 1, 1)
	l, err = gtk.LabelNew("Amount:")
	if err != nil {
		log.Fatal(err)
	}
	grid.Attach(l, 0, 2, 1, 1)

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
	remove.ConnectWithData("clicked", rmFn, ret)
	grid.Attach(remove, 2, 0, 1, 1)

	label, err := gtk.EntryNew()
	if err != nil {
		log.Fatal(err)
	}
	label.SetHExpand(true)
	ret.label = label
	grid.Attach(label, 1, 1, 2, 1)

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

	var iter gtk.TreeIter
	ls, err := gtk.ListStoreNew(glib.TYPE_STRING)
	if err != nil {
		log.Fatal(err)
	}
	ls.Append(&iter)
	choices := []string{"BTC", "mBTC", "μBTC"}
	s := make([]interface{}, len(choices))
	for i, v := range choices {
		s[i] = v
	}
	if err := ls.Set(&iter, []int{0}, []interface{}{"BTC"}); err != nil {
		fmt.Println(err)
	}
	ls.Append(&iter)
	if err := ls.Set(&iter, []int{0}, []interface{}{"mBTC"}); err != nil {
		fmt.Println(err)
	}
	ls.Append(&iter)
	if err := ls.Set(&iter, []int{0}, []interface{}{"μBTC"}); err != nil {
		fmt.Println(err)
	}
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

	grid.Attach(amounts, 1, 2, 1, 1)

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
	btn.Connect("clicked", func(ctx *glib.CallbackContext) {
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
			addr, err := r.payTo.GetText()
			if err != nil {
				d := errorDialog("Error getting payment address", err.Error())
				d.Run()
				d.Destroy()
				return
			}
			_, _, err = btcutil.DecodeAddress(addr)
			if err != nil {
				d := errorDialog("Invalid payment address",
					fmt.Sprintf("'%v' is not a valid payment address", addr))
				d.Run()
				d.Destroy()
				return
			}
			// TODO(jrick): confirm network is correct

			// Get amount and units and convert to float64
			amt := r.amount.GetValue()
			// TODO(jrick): constify these conversions
			switch r.combo.GetActive() {
			case 0: // BTC
				// nothing
			case 1: // mBTC
				amt /= 1000
			case 2: // uBTC
				amt /= 1000000
			}

			sendTo[addr] = amt
		}

		go func() {
			triggers.sendTx <- sendTo

			err := <-triggerReplies.sendTx
			if err != nil {
				glib.IdleAdd(func() {
					d := errorDialog("Unable to send transaction", err.Error())
					d.Run()
					d.Destroy()
				})
				return
			}
			// TODO(jrick): need to think about when to clear the entries.
			// Probably after the tx is validated and published?
			//recipients.Init()
		}()
	})
	SendCoins.SendBtn = submitBtn
	bot.Add(submitBtn)

	grid.Add(bot)

	return &grid.Container.Widget
}

func errorDialog(title, msg string) *gtk.MessageDialog {
	mDialog := gtk.MessageDialogNew(mainWindow, 0,
		gtk.MESSAGE_ERROR, gtk.BUTTONS_OK,
		msg)
	return mDialog
}
