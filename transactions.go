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
	"github.com/conformal/btcutil"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
	"strconv"
	"time"
)

type txDirection int

// Possible directions of a transaction.
const (
	Send txDirection = iota
	Recv
)

// String returns a transaction direction as a string.  Satisifies
// the fmt.Stringer interface.
func (d txDirection) String() string {
	switch d {
	case Send:
		return "Send"

	case Recv:
		return "Receive"

	default:
		return "Unknown"
	}
}

// TxAttributes holds the information that is shown by each transaction
// in the transactions view and overview pane.
type TxAttributes struct {
	Direction txDirection
	Address   string
	Amount    int64 // measured in satoshis
	Date      time.Time
}

var txWidgets struct {
	store    *gtk.ListStore
	treeview *gtk.TreeView
}

func createTransactions() *gtk.Widget {
	sw, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	store, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING,
		glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal(err)
	}
	tv, err := gtk.TreeViewNew()
	if err != nil {
		log.Fatal(err)
	}
	tv.SetModel(store)
	tv.SetHExpand(true)
	tv.SetVExpand(true)
	txWidgets.store = store
	txWidgets.treeview = tv
	sw.Add(tv)

	cr, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	col, err := gtk.TreeViewColumnNewWithAttribute("Date", cr, "text", 0)
	if err != nil {
		log.Fatal(err)
	}
	tv.AppendColumn(col)

	cr, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	col, err = gtk.TreeViewColumnNewWithAttribute("Type", cr, "text", 1)
	if err != nil {
		log.Fatal(err)
	}
	tv.AppendColumn(col)

	cr, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	col, err = gtk.TreeViewColumnNewWithAttribute("Address", cr, "text", 2)
	if err != nil {
		log.Fatal(err)
	}
	col.SetExpand(true)
	tv.AppendColumn(col)

	cr, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	col, err = gtk.TreeViewColumnNewWithAttribute("Amount", cr, "text", 3)
	if err != nil {
		log.Fatal(err)
	}
	tv.AppendColumn(col)

	return &sw.Bin.Container.Widget
}

func amountStr(amount int64) string {
	fAmount := float64(amount) / float64(btcutil.SatoshiPerBitcoin)
	return strconv.FormatFloat(fAmount, 'f', 8, 64)
}
