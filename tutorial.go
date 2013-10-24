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
)

// Using `` for string literals here is too messy, so concat "" strings.
const (
	welcomeText = "<b>Welcome to btcgui alpha!</b>\n" +
		"\n" +
		"The following steps will prepare you to begin using " +
		"btcgui.\n" +
		"\n" +
		"Click Next to open the next tip, or Close to skip the " +
		"rest of the tutorial.\n"

	disclaimerText = "<b>Disclaimer</b>\n" +
		"btcgui is still alpha-level development software and is " +
		"not yet ready to replace other Bitcoin wallet software.\n" +
		"\n" +
		"Running btcgui on the main Bitcoin network is currently " +
		"disabled until the software has matured.  Instead, btcgui " +
		"alpha operates on the Bitcoin test network (version 3). " +
		"Support for the main Bitcoin network will be enabled in " +
		"future versions when ready.\n" +
		"\n" +
		"btcgui is primarly written as a Unix GUI, and will be rough " +
		"around the edges on other platforms.  Development of native " +
		"applications for other platforms is planned for the future."

	connectText = "<b>Multiprocess Wallet Design</b>\n" +
		"\n" +
		"btcgui does not store your Bitcoin wallet or create and " +
		"send transactions, but instead connects to another program, " +
		"btcwallet, for wallet services.  Due to this design, btcgui " +
		"will not function if disconnected from btcwallet and almost " +
		"all btcgui features will be disabled.\n" +
		"\n" +
		"If btcgui is not currently connected to btcwallet, a " +
		"warning message will be displayed in the btcgui statusbar. " +
		"If this happens, first check that btcwallet is running and " +
		"not blocked behind a firewall.  If a connection can still " +
		"not be established, it's possible the wrong ports are being " +
		"used."

	createWalletText = "<b>Creating a Wallet</b>\n" +
		"\n" +
		"When first connecting a frontend such as btcgui to " +
		"btcwallet, a dialog will open asking for a wallet " +
		"passphrase.  btcwallet does not support wallets with " +
		"unencrypted private keys, and will not autogenerate wallets " +
		"for this reason.\n" +
		"\n" +
		"To create this wallet, enter and repeat a passphrase in " +
		"the passphrase dialog and press OK.  btcwallet will create " +
		"the encrypted wallet and begin notifying btcgui of changes " +
		"to the wallet, such as new account balances."

	receiveText = "<b>Receiving Funds</b>\n" +
		"\n" +
		"Bitcoins can be received by giving a payment address from " +
		"your wallet to other Bitcoin users.  Payment addresses can " +
		"be managed under the \"Receive Coins\" tab.  From this " +
		"view, all addresses for your wallet can be viewed and " +
		"copied to your clipboard. New addresses can also be " +
		"generated if you do not wish to reuse an older address.\n" +
		"\n" +
		"If neither you nor anyone you know has testnet Bitcoins, " +
		"an online Bitcoin testnet faucet can be used to receive " +
		"testnet coins."

	sendText = "<b>Sending Funds</b>\n" +
		"\n" +
		"To send Bitcoins to others, open the \"Send Coins\" tab. " +
		"Enter the payment addresses and Bitcoin amounts to send " +
		"for one or more recipient (adding additional recipients as " +
		"needed).  A wallet must be unlocked before new " +
		"transactions can be created."

	futureFeaturesText = "<b>Future Features</b>\n" +
		"\n" +
		"btcgui alpha is still under heavy development and is " +
		"missing features needed for an everyday wallet " +
		"application.\n" +
		"\n" +
		"Features like displaying transactions sent to and from " +
		"owned addresses, saving others' addresses to an address " +
		"book, and multiple account support are planned for future " +
		"versions."

	feedbackText = "<b>Feedback is appreciated!</b>\n" +
		"\n" +
		"Stop by Conformal's " +
		"<a href=\"https://opensource.conformal.com/wiki/IRC_server\">IRC server</a> " +
		"(channel #btcd) to let us know what you think!"
)

// dialogMessages holds the dialog messages successively shown in the
// tutorial dialog.
var dialogMessages = []string{
	welcomeText,
	disclaimerText,
	connectText,
	createWalletText,
	receiveText,
	sendText,
	futureFeaturesText,
	feedbackText,
}

// CreateTutorialDialog opens a tutorial dialog explaining usage for a
// first-time user.  If appWindow is non-nil, it will be used as the
// parent window of the dialog.  If nil, the tutorial dialog will open as
// a top-level window and a new application main window will be created
// and opened after the final tutorial message is shown.
func CreateTutorialDialog(appWindow *gtk.Window) (*gtk.Dialog, error) {
	d, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	d.SetTitle("btcgui First Start Tutorial")
	box, err := d.GetContentArea()
	if err != nil {
		return nil, err
	}
	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	box.Add(grid)

	d.SetDefaultGeometry(500, 100)

	if appWindow != nil {
		d.SetTransientFor(appWindow)
		d.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	} else {
		d.Connect("destroy", func() {
			go StartMainApplication()
		})
		d.SetPosition(gtk.WIN_POS_CENTER)
	}

	// Add a notebook and tab for each dialog message.
	nb, err := gtk.NotebookNew()
	if err != nil {
		return nil, err
	}

	// Because the labels added below will wrap and their final minimum
	// heights and widths will be absurdly large, first give a size request
	// and show the notebook (allocating space for the requested size).
	// This will make text wrapping labels size nicely inside the notebook.
	nb.SetSizeRequest(500, 100)
	nb.Show()

	// Create messages and append each in a notebook page.
	for _, msg := range dialogMessages {
		lbl, err := gtk.LabelNew("")
		if err != nil {
			return nil, err
		}
		lbl.SetMarkup(msg)
		lbl.SetLineWrap(true)
		lbl.Show()
		lbl.SetAlignment(0, 0)
		nb.AppendPage(lbl, nil)
	}
	nb.SetShowTabs(false)
	grid.Add(nb)

	prevPage, err := d.AddButton("_Previous", gtk.RESPONSE_NONE)
	if err != nil {
		return nil, err
	}
	prevPage.SetSensitive(false)
	nextPage, err := d.AddButton("_Next", gtk.RESPONSE_NONE)
	if err != nil {
		return nil, err
	}
	prevPage.Connect("clicked", func() {
		nb.PrevPage()
		pagen := nb.GetCurrentPage()
		if pagen == 0 {
			prevPage.SetSensitive(false)
		}
		nextPage.SetSensitive(true)
	})
	nextPage.Connect("clicked", func() {
		nb.NextPage()
		pagen := nb.GetCurrentPage()
		if pagen == len(dialogMessages)-1 {
			nextPage.SetSensitive(false)
		}
		prevPage.SetSensitive(true)
	})
	_, err = d.AddButton("_Close", gtk.RESPONSE_CLOSE)
	if err != nil {
		return nil, err
	}
	d.Connect("response", func(cxt *glib.CallbackContext) {
		response := gtk.ResponseType(int32(cxt.Arg(0)))
		switch response {
		case gtk.RESPONSE_CLOSE:
			// Using w.Close() would be nice but it needs GTK 3.10.
			d.Hide()
			d.Destroy()
		}
	})

	return d, nil
}
