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
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcutil"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
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
	Amount    btcutil.Amount
	Date      time.Time
}

func NewTxAttributesFromJSON(r *btcjson.ListTransactionsResult) (*TxAttributes, error) {
	var direction txDirection
	switch r.Category {
	case "send":
		direction = Send

	case "receive":
		direction = Recv

	default: // TODO: support additional listtransaction categories.
		return nil, fmt.Errorf("unsupported tx category: %v", r.Category)
	}

	amount, err := btcutil.NewAmount(r.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	return &TxAttributes{
		Direction: direction,
		Address:   r.Address,
		Amount:    amount,
		Date:      time.Unix(r.TimeReceived, 0),
	}, nil
}

// TODO(jrick): This must be removed.  It is only being kept around because
// *all* responses are currently being sent as the nasty types determined
// by encoding/json instead of the correct btcjson result type.
func NewTxAttributesFromMap(m map[string]interface{}) (*TxAttributes, error) {
	var direction txDirection
	category, ok := m["category"].(string)
	if !ok {
		return nil, errors.New("unspecified category")
	}
	switch category {
	case "send":
		direction = Send

	case "receive":
		direction = Recv

	default: // TODO: support additional listtransaction categories.
		return nil, fmt.Errorf("unsupported tx category: %v", category)
	}

	address, ok := m["address"].(string)
	if !ok {
		return nil, errors.New("unspecified address")
	}

	famount, ok := m["amount"].(float64)
	if !ok {
		return nil, errors.New("unspecified amount")
	}
	amount, err := btcutil.NewAmount(famount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %v", err)
	}

	funixDate, ok := m["timereceived"].(float64)
	if !ok {
		return nil, errors.New("unspecified time")
	}
	if fblockTime, ok := m["blocktime"].(float64); ok {
		if fblockTime < funixDate {
			funixDate = fblockTime
		}
	}
	unixDate := int64(funixDate)

	return &TxAttributes{
		Direction: direction,
		Address:   address,
		Amount:    amount,
		Date:      time.Unix(unixDate, 0),
	}, nil
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
