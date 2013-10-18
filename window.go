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
	"github.com/conformal/gotk3/gtk"
)

var (
	mainWindow *gtk.Window
)

// CreateWindow creates the toplevel window for the GUI.
func CreateWindow() (*gtk.Window, error) {
	var err error
	mainWindow, err = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		return nil, err
	}
	mainWindow.SetTitle("btcgui")
	mainWindow.Connect("destroy", func() {
		gtk.MainQuit()
	})

	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	grid.Add(createMenuBar())

	notebook, err := gtk.NotebookNew()
	if err != nil {
		return nil, err
	}
	notebook.SetHExpand(true)
	notebook.SetVExpand(true)
	grid.Add(notebook)

	l, err := gtk.LabelNew("Overview")
	if err != nil {
		return nil, err
	}
	notebook.AppendPage(createOverview(), l)

	l, err = gtk.LabelNew("Send Coins")
	if err != nil {
		return nil, err
	}
	notebook.AppendPage(createSendCoins(), l)

	l, err = gtk.LabelNew("Receive Coins")
	if err != nil {
		return nil, err
	}
	notebook.AppendPage(createRecvCoins(), l)

	// TODO(jrick): Add back when transaction list is implemented.
	/*
		l, err = gtk.LabelNew("Transactions")
		if err != nil {
			log.Fatal(err)
		}
		notebook.AppendPage(createTransactions(), l)
	*/

	// TODO(jrick): Add back when address book is implemented.
	/*
		l, err = gtk.LabelNew("Address Book")
		if err != nil {
			log.Fatal(err)
		}
		notebook.AppendPage(createAddrBook(), l)
	*/

	grid.Add(createStatusbar())

	mainWindow.Add(grid)

	mainWindow.SetDefaultGeometry(800, 600)

	return mainWindow, nil
}
