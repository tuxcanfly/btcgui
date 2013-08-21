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
	"time"
)

var txWidgets struct {
	store    *gtk.ListStore
	treeview *gtk.TreeView
}

func createTransactions() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

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
	grid.Add(tv)

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

	// some example addresses
	var iterPurchase gtk.TreeIter
	store.Append(&iterPurchase)
	const layout = "01/02/2006"
	store.Set(&iterPurchase, []int{0, 1, 2, 3}, []interface{}{
		time.Now().Format(layout), "Purchase", "01234567890",
		"0.50000000"})
	var iterPayment gtk.TreeIter
	store.Append(&iterPayment)
	store.Set(&iterPayment, []int{0, 1, 2, 3}, []interface{}{
		time.Now().Format(layout), "Payment", "0987654321",
		"0.50000000"})

	return &grid.Container.Widget
}
