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
	"github.com/conformal/go-flags"
	"github.com/conformal/gotk3/gtk"
	"log"
	"time"
)

type options struct {
	User     string `short:"u" long:"user" description:"rpc username"`
	Password string `short:"p" long:"password" description:"rpc password"`
	Server   string `short:"s" long:"server" description:"rpc server address and port"`
}

var (
	// Defaults
	opts = options{
		Server: "127.0.0.1:8332",
	}
)

func main() {
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		log.Fatal("Could not read cmd line options.", err)
	}

	gtk.Init(nil)

	w := CreateWindow()
	w.SetDefaultSize(800, 600)
	w.ShowAll()

	// Listen for updates and update GUI with new info.  Attempt
	// reconnect if connection is lost or cannot be established.
	go func() {
		for {
			replies := make(chan error)
			done := make(chan int)
			go func() {
				ListenAndUpdate(replies)
				close(done)
			}()
		selectLoop:
			for {
				select {
				case <-done:
					break selectLoop
				case err := <-replies:
					switch err {
					case ErrConnectionRefused:
						time.Sleep(5 * time.Second)
					case ErrConnectionLost:
						time.Sleep(5 * time.Second)
					case nil:
						// connected
					default:
						log.Printf("Unknown connect error: %v", err)
					}
				}
			}
		}
	}()

	gtk.Main()
}
