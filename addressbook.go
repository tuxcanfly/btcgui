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
	"github.com/conformal/gotk3/gdk"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
)

var addrBookWidgets struct {
	store    *gtk.ListStore
	treeview *gtk.TreeView
}

func createAddrBook() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	store, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal(err)
	}
	tv, err := gtk.TreeViewNew()
	if err != nil {
		log.Fatal(err)
	}
	tv.SetModel(store)
	addrBookWidgets.store = store
	addrBookWidgets.treeview = tv

	sw, err := gtk.ScrolledWindowNew(nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	sw.Add(tv)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetHExpand(true)
	sw.SetVExpand(true)
	grid.Add(sw)

	renderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	renderer.Set("editable", true)
	renderer.Set("editable-set", true)
	renderer.Connect("edited", func(_ *glib.Object, path, text string) {
		iter, err := store.GetIterFromString(path)
		if err == nil {
			store.Set(iter, []int{0}, []interface{}{text})
		}
	})

	col, err := gtk.TreeViewColumnNewWithAttribute("Label", renderer, "text", 0)
	if err != nil {
		log.Fatal(err)
	}
	col.SetExpand(true)
	tv.AppendColumn(col)

	renderer, err = gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal(err)
	}
	renderer.Set("editable", true)
	renderer.Set("editable-set", true)
	renderer.Connect("edited", func(_ *glib.Object, path, text string) {
		iter, err := store.GetIterFromString(path)
		if err == nil {
			// TODO(jrick): verify this is a valid address
			store.Set(iter, []int{1}, []interface{}{text})
		}
	})
	col, err = gtk.TreeViewColumnNewWithAttribute("Address", renderer,
		"text", 1)
	if err != nil {
		log.Fatal(err)
	}
	col.SetMinWidth(350)
	tv.AppendColumn(col)

	// put in an example address
	iter := store.Append()
	store.Set(iter, []int{0, 1}, []interface{}{"editable label", "01234567890"})

	buttons, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}

	newAddr, err := gtk.ButtonNewWithLabel("New Address")
	if err != nil {
		log.Fatal(err)
	}
	newAddr.SetSizeRequest(150, -1)
	newAddr.Connect("clicked", func() {
		iter := store.Append()
		store.Set(iter, []int{0, 1}, []interface{}{"", "new address"})
	})
	buttons.Add(newAddr)

	cpyAddr, err := gtk.ButtonNewWithLabel("Copy Address")
	if err != nil {
		log.Fatal(err)
	}
	cpyAddr.SetSizeRequest(150, -1)
	cpyAddr.Connect("clicked", func() {
		sel, err := tv.GetSelection()
		if err != nil {
			log.Fatal(err)
		}
		var iter gtk.TreeIter
		if sel.GetSelected(nil, &iter) {
			val, err := store.GetValue(&iter, 1)
			if err != nil {
				log.Fatal(err)
			}
			display, err := gdk.DisplayGetDefault()
			if err != nil {
				log.Fatal(err)
			}

			clipboard, err := gtk.ClipboardGetForDisplay(
				display,
				gdk.SELECTION_CLIPBOARD)
			if err != nil {
				log.Fatal(err)
			}
			primary, err := gtk.ClipboardGetForDisplay(
				display,
				gdk.SELECTION_PRIMARY)
			if err != nil {
				log.Fatal(err)
			}
			s, _ := val.GetString()
			clipboard.SetText(s)
			primary.SetText(s)
		}
	})
	buttons.Add(cpyAddr)
	grid.Add(buttons)

	return &grid.Container.Widget
}
