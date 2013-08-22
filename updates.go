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
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
	"math"
	"strconv"
	"sync"
)

const (
	blocksForConfirmation = 6
	satoshiPerBTC         = 100000000
)

// Errors
var (
	ConnectionLost = errors.New("Connection lost.")
)

var (
	// Sequence number for id field of json messages and replies.  Uses a
	// mutex for synchronization as it is used by multiple goroutines.
	seq = struct {
		sync.Mutex
		n uint64
	}{}

	// replyHandlers maps between a uint64 sequence id for json
	// messages and replies, and a function to handle the returned
	// result.  Mutex protects against multiple writes.
	replyHandlers = struct {
		sync.Mutex
		m map[uint64]func(interface{}, interface{})
	}{
		m: make(map[uint64]func(interface{}, interface{})),
	}

	// Channels filled from fetchFuncs and read by updateFuncs.
	updateChans = struct {
		btcdConnected    chan int
		btcdDisconnected chan int
		balance          chan float64
		unconfirmed      chan float64
		bcHeight         chan int64
		bcHeightRemote   chan int64
		addrs            chan []string
		lockState        chan bool
	}{
		btcdConnected:    make(chan int),
		btcdDisconnected: make(chan int),
		balance:          make(chan float64),
		unconfirmed:      make(chan float64),
		bcHeight:         make(chan int64),
		bcHeightRemote:   make(chan int64),
		addrs:            make(chan []string),
		lockState:        make(chan bool),
	}

	triggers = struct {
		newAddr      chan int
		lockWallet   chan int
		unlockWallet chan *UnlockParams
	}{
		newAddr:      make(chan int),
		lockWallet:   make(chan int),
		unlockWallet: make(chan *UnlockParams),
	}

	triggerReplies = struct {
		unlockSuccessful chan bool
	}{
		unlockSuccessful: make(chan bool),
	}

	reqFuncs = [](func(*websocket.Conn) error){
		reqAddresses,
		reqBalance,
		reqProgress,
		//reqRemoteProgress,
		reqUnconfirmed,
		reqLockState,
	}
	updateFuncs = [](func()){
		updateAddresses,
		updateBalance,
		updateConnectionState,
		updateLockState,
		updateProgress,
		updateUnconfirmed,
	}
)

func ListenAndUpdate() error {
	// Connect to websocket.
	// TODO(jrick): don't hardcode port
	// TODO(jrick): use TLS
	// TODO(jrick): http username/password?
	ws, err := websocket.Dial("ws://localhost:8332/frontend", "",
		"http://localhost/")
	if err != nil {
		return err
	}

	// Start updater funcs
	for _, f := range updateFuncs {
		go f()
	}

	// Channel for replies and notifications from btcwallet.
	replies := make(chan []byte, 100)

	go func() {
		for {
			// Receive message from wallet
			var msg []byte
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				close(replies)
				return
			}
			replies <- msg
		}
	}()

	for _, f := range reqFuncs {
		// TODO(jrick): don't throw away errors here
		go f(ws)
	}

	for {
		select {
		case r, ok := <-replies:
			if !ok {
				// btcwallet connection lost.
				return ConnectionLost
			}
			var rply map[string]interface{}
			json.Unmarshal(r, &rply)

			// json.Unmarshal unmarshalls numbers as float64
			floatId, ok := rply["id"].(float64)
			if ok {
				uintId := uint64(floatId)
				replyHandlers.Lock()
				f := replyHandlers.m[uintId]
				delete(replyHandlers.m, uintId)
				replyHandlers.Unlock()
				if f != nil {
					go f(rply["result"], rply["error"])
				}
			} else if strId, ok := (rply["id"]).(string); ok {
				// Handle btcwallet notification.
				go handleBtcwalletNtfn(strId, rply["result"])
			}
		case <-triggers.newAddr:
			go reqNewAddr(ws)
		case <-triggers.lockWallet:
			go cmdWalletLock(ws)
		case params := <-triggers.unlockWallet:
			go cmdWalletPassphrase(ws, params)
		}
	}
}

// handleBtcwalletNtfn processes notifications from btcwallet and
// btcd, triggering the GUI updates associated with the notification.
func handleBtcwalletNtfn(id string, result interface{}) {
	switch id {
	case "btcwallet:btcconnected":
		updateChans.btcdConnected <- 1
	case "btcwallet:btcddisconnected":
		updateChans.btcdDisconnected <- 1
	case "btcwallet:newbalance":
	case "btcwallet:newwalletlockstate":
		if r, ok := result.(bool); ok {
			updateChans.lockState <- r
		}
	case "btcd:newblockchainheight":
		if r, ok := result.(float64); ok {
			updateChans.bcHeight <- int64(r)
		}
	default:
		log.Printf("Unhandled message with id '%s'\n", id)
	}
}

// reqNewAddr requests a new wallet address.
//
// TODO(jrick): support addresses other than the default address.
func reqNewAddr(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	msg, err := btcjson.CreateMessageWithId("getnewaddress", n, "")
	if err != nil {
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		glib.IdleAdd(func() {
			var iter gtk.TreeIter
			RecvElems.Store.Append(&iter)
			RecvElems.Store.Set(&iter, []int{0, 1},
				[]interface{}{"", result.(string)})
		})
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

// reqProgress requests the blockchain height.
func reqProgress(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	msg, err := btcjson.CreateMessageWithId("getblockcount", n)
	if err != nil {
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		if r, ok := result.(float64); ok {
			updateChans.bcHeight <- int64(r)
		}
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

/* TODO(jrick): is this being done with websockets?
// Connect to a remote server that should be up to date to compare
// blochchain height progress.  Let's not hard code this server in the
// future.
func fetchRemoteProgress(ws *websocket.Conn) {
	defer wg.Done()

	msg, err := btcjson.CreateMessage("getblockcount")
	if err != nil {
		log.Print(err)
		updateChans.bcHeightRemote <- -1
		return
	}
	remote, err := btcjson.RpcCommand("bitcoinrpc",
		"8X3yBH4oBq98wzVubaMhQrw5ZJWMECf5e2WoKz5DzBf",
		"10.168.0.17:8332",
		msg)
	if err != nil {
		log.Print(err)
		updateChans.bcHeightRemote <- -1
		return
	}

	rmtResult, err := btcjson.JSONToAmount(remote.Result.(float64))
	if err != nil {
		log.Print(err)
		updateChans.bcHeightRemote <- -1
		return
	}
	updateChans.bcHeightRemote <- rmtResult / satoshiPerBTC
}
*/

// reqAddresses requestse all addresses for an account.
//
// TODO(jrick): support addresses other than the default address.
func reqAddresses(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	msg, err := btcjson.CreateMessageWithId("getaddressesbyaccount", n, "")
	if err != nil {
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		if r, ok := result.([]interface{}); ok {
			addrs := []string{}
			for _, v := range r {
				addrs = append(addrs, v.(string))
			}
			updateChans.addrs <- addrs
		}
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

// reqBalance requests the current balance for an account.
//
// TODO(jrick): support addresses other than the default address.
func reqBalance(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	msg, err := btcjson.CreateMessageWithId("getbalance", n, "", blocksForConfirmation)
	if err != nil {
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		if r, ok := result.(float64); ok {
			updateChans.balance <- r
		}
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

// reqBalance requests the current unconfirmed balance for an account.
//
// TODO(jrick): support addresses other than the default address.
func reqUnconfirmed(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	msg, err := btcjson.CreateMessageWithId("getbalance", n, "", 0)
	if err != nil {
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		if r, ok := result.(float64); ok {
			updateChans.balance <- r
		}
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

func reqLockState(ws *websocket.Conn) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	m := btcjson.Message{
		Jsonrpc: "",
		Id:      n,
		Method:  "walletislocked",
		Params:  []interface{}{},
	}
	msg, _ := json.Marshal(&m)

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		if r, ok := result.(bool); ok {
			updateChans.lockState <- r
		}
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

func cmdWalletLock(ws *websocket.Conn) error {
	// Don't really care about handling replies.  If wallet is already
	// locked, great.
	msg, err := btcjson.CreateMessage("walletlock")
	if err != nil {
		return err
	}

	return websocket.Message.Send(ws, msg)
}

func cmdWalletPassphrase(ws *websocket.Conn, params *UnlockParams) error {
	seq.Lock()
	n := seq.n
	seq.n++
	seq.Unlock()

	m := btcjson.Message{
		Jsonrpc: "",
		Id:      n,
		Method:  "walletpassphrase",
		Params: []interface{}{
			params.passphrase,
			params.timeout,
		},
	}
	msg, _ := json.Marshal(&m)

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result, err interface{}) {
		triggerReplies.unlockSuccessful <- err == nil
	}
	replyHandlers.Unlock()

	return websocket.Message.Send(ws, msg)
}

// strSliceEqual checks if each string in a is equal to each string in b.
func strSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// updateConnectionState listens for connection status changes to btcd
// and btcwallet, updating the GUI when necessary.
func updateConnectionState() {
	for {
		select {
		case <-updateChans.btcdConnected:
			glib.IdleAdd(func() {
				StatusElems.Lab.SetText("Established connection to btcd.")
				StatusElems.Pb.Hide()
			})
		case <-updateChans.btcdDisconnected:
			glib.IdleAdd(func() {
				StatusElems.Lab.SetText("Connection to btcd lost.")
				StatusElems.Pb.Hide()
			})
		}
	}
}

// updateAddresses listens for new wallet addresses, updating the GUI when
// necessary.
func updateAddresses() {
	for {
		addrs := <-updateChans.addrs
		glib.IdleAdd(func() {
			RecvElems.Store.Clear()
		})
		for i := range addrs {
			addr := addrs[i]
			glib.IdleAdd(func() {
				var iter gtk.TreeIter
				RecvElems.Store.Append(&iter)
				RecvElems.Store.Set(&iter, []int{1},
					[]interface{}{addr})
			})
		}
	}
}

// updateBalance listens for new wallet account balances, updating the GUI
// when necessary.
func updateBalance() {
	for {
		balance, ok := <-updateChans.balance
		if !ok {
			return
		}

		var s string
		if math.IsNaN(balance) {
			s = "unknown"
		} else {
			s = strconv.FormatFloat(balance, 'f', 8, 64) + " BTC"
		}
		glib.IdleAdd(func() {
			Overview.Balance.SetMarkup("<b>" + s + "</b>")
			SendCoins.Balance.SetText("Balance: " + s)
		})
	}
}

// updateBalance listens for new wallet account unconfirmed balances, updating
// the GUI when necessary.
func updateUnconfirmed() {
	for {
		unconfirmed, ok := <-updateChans.unconfirmed
		if !ok {
			return
		}

		var s string
		if math.IsNaN(unconfirmed) {
			s = "unknown"
		} else {
			balStr := strconv.FormatFloat(unconfirmed, 'f', 8, 64) + " BTC"
			s = "<b>" + balStr + "</b>"
		}
		glib.IdleAdd(func() {
			Overview.Unconfirmed.SetMarkup(s)
		})
	}
}

func updateLockState() {
	for {
		locked, ok := <-updateChans.lockState
		if !ok {
			return
		}

		if locked {
			glib.IdleAdd(func() {
				MenuBar.Settings.Lock.SetSensitive(false)
				MenuBar.Settings.Unlock.SetSensitive(true)
			})
		} else {
			glib.IdleAdd(func() {
				MenuBar.Settings.Lock.SetSensitive(true)
				MenuBar.Settings.Unlock.SetSensitive(false)
			})
		}
	}
}

// XXX spilt this?
func updateProgress() {
	for {
		bcHeight, ok := <-updateChans.bcHeight
		if !ok {
			return
		}

		// TODO(jrick) this can go back when remote height is updated.
		/*
			bcHeightRemote, ok := <-updateChans.bcHeightRemote
			if !ok {
				return
			}

			if bcHeight >= 0 && bcHeightRemote >= 0 {
				percentDone := float64(bcHeight) / float64(bcHeightRemote)
				if percentDone < 1 {
					s := fmt.Sprintf("%d of ~%d blocks", bcHeight,
						bcHeightRemote)
					glib.IdleAdd(StatusElems.Lab.SetText,
						"Updating blockchain...")
					glib.IdleAdd(StatusElems.Pb.SetText, s)
					glib.IdleAdd(StatusElems.Pb.SetFraction, percentDone)
					glib.IdleAdd(StatusElems.Pb.Show)
				} else {
					s := fmt.Sprintf("%d blocks", bcHeight)
					glib.IdleAdd(StatusElems.Lab.SetText, s)
					glib.IdleAdd(StatusElems.Pb.Hide)
				}
			} else if bcHeight >= 0 && bcHeightRemote == -1 {
				s := fmt.Sprintf("%d blocks", bcHeight)
				glib.IdleAdd(StatusElems.Lab.SetText, s)
				glib.IdleAdd(StatusElems.Pb.Hide)
			} else {
				glib.IdleAdd(StatusElems.Lab.SetText,
					"Error getting blockchain height")
				glib.IdleAdd(StatusElems.Pb.Hide)
			}
		*/

		s := fmt.Sprintf("%d blocks", bcHeight)
		glib.IdleAdd(func() {
			StatusElems.Lab.SetText(s)
			StatusElems.Pb.Hide()
		})
	}
}
