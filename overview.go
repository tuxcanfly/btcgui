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
	"fmt"
	"github.com/conformal/gotk3/gtk"
	"log"
)

const NOverviewTxs = 5

var (
	// Overview holds pointers to widgets shown in the overview tab.
	Overview = struct {
		Balance       *gtk.Label
		Unconfirmed   *gtk.Label
		NTransactions *gtk.Label // TODO(jrick): update with value from btcwallet, requires extension.
		Txs           *gtk.Grid
		TxList        []*gtk.Widget
	}{
		TxList: make([]*gtk.Widget, 0, NOverviewTxs),
	}

	// Holds pointers to the latest tx label widgets.
)

func createWalletInfo() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}

	header, err := gtk.LabelNew("")
	if err != nil {
		log.Fatal(err)
	}
	header.SetMarkup("<b>Wallet</b>")
	header.OverrideFont("sans-serif 16")
	header.SetHAlign(gtk.ALIGN_START)
	grid.Attach(header, 0, 0, 1, 1)

	balance, err := gtk.LabelNew("Balance:")
	if err != nil {
		log.Fatal(err)
	}
	balance.SetHAlign(gtk.ALIGN_START)
	grid.Attach(balance, 0, 1, 1, 1)

	unconfirmed, err := gtk.LabelNew("Unconfirmed:")
	if err != nil {
		log.Fatal(err)
	}
	unconfirmed.SetHAlign(gtk.ALIGN_START)
	grid.Attach(unconfirmed, 0, 2, 1, 1)

	balance, err = gtk.LabelNew("")
	if err != nil {
		log.Fatal(err)
	}
	balance.SetHAlign(gtk.ALIGN_START)
	grid.Attach(balance, 1, 1, 1, 1)
	Overview.Balance = balance

	unconfirmed, err = gtk.LabelNew("")
	if err != nil {
		log.Fatal(err)
	}
	grid.Attach(unconfirmed, 1, 2, 1, 1)
	Overview.Unconfirmed = unconfirmed

	/*
		transactions, err := gtk.LabelNew("Number of transactions:")
		if err != nil {
			log.Fatal(err)
		}
		transactions.SetHAlign(gtk.ALIGN_START)
		grid.Attach(transactions, 0, 3, 1, 1)

		transactions, err = gtk.LabelNew("a lot")
		if err != nil {
			log.Fatal(err)
		}
		transactions.SetHAlign(gtk.ALIGN_START)
		grid.Attach(transactions, 1, 3, 1, 1)
		Overview.NTransactions = transactions
	*/

	return &grid.Container.Widget
}

func createTxInfo() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)

	l, err := gtk.LabelNew("")
	if err != nil {
		log.Fatal(err)
	}
	l.SetMarkup("<b>Recent Transactions</b>")
	l.OverrideFont("sans-serif 10")
	l.SetHAlign(gtk.ALIGN_START)
	grid.Add(l)

	txGrid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}
	txGrid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	grid.Add(txGrid)

	Overview.Txs = txGrid

	return &grid.Container.Widget
}

func createTxLabel(attr *TxAttributes) (*gtk.Widget, error) {
	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetHExpand(true)

	var amtLabel *gtk.Label
	var description *gtk.Label
	var icon *gtk.Image
	switch attr.Direction {
	case Send:
		amtLabel, err = gtk.LabelNew(amountStr(attr.Amount))
		if err != nil {
			return nil, err
		}

		description, err = gtk.LabelNew(fmt.Sprintf("Send (%s)", attr.Address))
		if err != nil {
			return nil, err
		}

		icon, err = gtk.ImageNewFromIconName("go-next",
			gtk.ICON_SIZE_SMALL_TOOLBAR)
		if err != nil {
			return nil, err
		}

	case Recv:
		amtLabel, err = gtk.LabelNew(amountStr(attr.Amount))
		if err != nil {
			return nil, err
		}

		description, err = gtk.LabelNew(fmt.Sprintf("Receive (%s)", attr.Address))
		if err != nil {
			return nil, err
		}

		icon, err = gtk.ImageNewFromIconName("go-previous",
			gtk.ICON_SIZE_SMALL_TOOLBAR)
		if err != nil {
			return nil, err
		}
	}
	grid.Attach(icon, 0, 0, 2, 2)
	grid.Attach(description, 2, 1, 2, 1)
	description.SetHAlign(gtk.ALIGN_START)
	grid.Attach(amtLabel, 3, 0, 1, 1)
	amtLabel.SetHAlign(gtk.ALIGN_END)
	amtLabel.SetHExpand(true)

	date, err := gtk.LabelNew(attr.Date.Format("Jan 2, 2006 at 3:04 PM"))
	if err != nil {
		return nil, err
	}
	grid.Attach(date, 2, 0, 1, 1)
	date.SetHAlign(gtk.ALIGN_START)

	grid.SetHAlign(gtk.ALIGN_FILL)

	return &grid.Container.Widget, nil
}

func createOverview() *gtk.Widget {
	grid, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}

	grid.SetColumnHomogeneous(true)
	grid.Add(createWalletInfo())
	grid.Add(createTxInfo())

	return &grid.Container.Widget
}
