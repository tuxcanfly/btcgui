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

const txFeeMessage = "Optional transaction fee to help make sure transactions are processed quickly."

func createTxFeeDialog() (*gtk.Dialog, error) {
	dialog, err := gtk.DialogNew()
	if err != nil {
		return nil, err
	}
	dialog.SetTitle("Set Transaction Fee")

	dialog.AddButton("_OK", gtk.RESPONSE_OK)
	dialog.AddButton("_Cancel", gtk.RESPONSE_CANCEL)

	grid, err := gtk.GridNew()
	if err != nil {
		return nil, err
	}
	grid.SetHExpand(true)
	grid.SetVExpand(true)
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	b, err := dialog.GetContentArea()
	if err != nil {
		return nil, err
	}
	b.Add(grid)
	b.SetHExpand(true)
	b.SetVExpand(true)

	l, err := gtk.LabelNew(txFeeMessage)
	if err != nil {
		return nil, err
	}
	l.SetHExpand(true)
	l.SetVExpand(true)
	l.SetHAlign(gtk.ALIGN_START)
	grid.Add(l)

	spinb, err := gtk.SpinButtonNewWithRange(0, 21000000, 0.00000001)
	if err != nil {
		return nil, err
	}
	grid.Add(spinb)

	dialog.SetTransientFor(mainWindow)
	dialog.SetPosition(gtk.WIN_POS_CENTER_ON_PARENT)
	dialog.ShowAll()

	dialog.Connect("response", func(_ *glib.Object, rt gtk.ResponseType) {
		switch rt {
		case gtk.RESPONSE_OK:
			fee := spinb.GetValue()
			go func() {
				triggers.setTxFee <- fee

				if err := <-triggerReplies.setTxFeeErr; err != nil {
					d := errorDialog("Error setting transaction fee:",
						err.Error())
					d.Run()
					d.Destroy()
				} else {
					glib.IdleAdd(func() {
						dialog.Destroy()
					})
				}
			}()

		case gtk.RESPONSE_CANCEL:
			dialog.Destroy()
		}
	})

	return dialog, nil
}
