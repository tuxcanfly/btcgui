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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcjson/btcws"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/go-socks/socks"
	"github.com/btcsuite/websocket"
	"github.com/conformal/gotk3/glib"
)

const (
	blocksForConfirmation = 6
	satoshiPerBTC         = 100000000
)

// Errors
var (
	// ErrConnectionRefused describes an error where a connection to
	// another process was refused.
	ErrConnectionRefused = errors.New("connection refused")

	// ErrConnectionLost describes an error where a connection to
	// another process was lost.
	ErrConnectionLost = errors.New("connection lost")
)

var (
	// NewJSONID is used to receive the next unique JSON ID for
	// btcwallet requests, starting from zero and incrementing by one
	// after each read.
	NewJSONID = make(chan uint64)

	// replyHandlers maps between a uint64 sequence id for json
	// messages and replies, and a function to handle the returned
	// result.  Mutex protects against multiple writes.
	replyHandlers = struct {
		sync.RWMutex
		m map[uint64]func(interface{}, *btcjson.Error)
	}{
		m: make(map[uint64]func(interface{}, *btcjson.Error)),
	}

	// Channels filled from fetchFuncs and read by updateFuncs.
	updateChans = struct {
		addrs              chan []string
		balance            chan btcutil.Amount
		btcdConnected      chan bool
		btcwalletConnected chan bool
		bcHeight           chan int32
		bcHeightRemote     chan int32
		lockState          chan bool
		unconfirmed        chan btcutil.Amount
		appendTx           chan *TxAttributes
		prependTx          chan *TxAttributes
		appendOverviewTx   chan *TxAttributes
		prependOverviewTx  chan *TxAttributes
	}{
		addrs:              make(chan []string),
		balance:            make(chan btcutil.Amount),
		btcdConnected:      make(chan bool),
		btcwalletConnected: make(chan bool),
		bcHeight:           make(chan int32),
		bcHeightRemote:     make(chan int32),
		lockState:          make(chan bool),
		unconfirmed:        make(chan btcutil.Amount),
		appendTx:           make(chan *TxAttributes),
		prependTx:          make(chan *TxAttributes),
		appendOverviewTx:   make(chan *TxAttributes),
		prependOverviewTx:  make(chan *TxAttributes),
	}

	triggers = struct {
		newAddr      chan int
		newWallet    chan *NewWalletParams
		lockWallet   chan int
		unlockWallet chan *UnlockParams
		sendTx       chan map[string]float64
		setTxFee     chan float64
	}{
		newAddr:      make(chan int),
		newWallet:    make(chan *NewWalletParams),
		lockWallet:   make(chan int),
		unlockWallet: make(chan *UnlockParams),
		sendTx:       make(chan map[string]float64),
		setTxFee:     make(chan float64),
	}

	triggerReplies = struct {
		newAddr           chan interface{}
		unlockSuccessful  chan bool
		walletCreationErr chan error
		sendTx            chan error
		setTxFeeErr       chan error
	}{
		newAddr:           make(chan interface{}),
		unlockSuccessful:  make(chan bool),
		walletCreationErr: make(chan error),
		sendTx:            make(chan error),
		setTxFeeErr:       make(chan error),
	}

	walletReqFuncs = []func(*websocket.Conn){
		cmdGetAddressesByAccount,
		cmdGetBalance,
		cmdGetBlockCount,
		cmdGetUnconfirmedBalance,
		cmdListAllTransactions,
		cmdWalletIsLocked,
	}
	updateFuncs = [](func()){
		updateAddresses,
		updateBalance,
		updateConnectionState,
		updateLockState,
		updateProgress,
		updateTransactions,
		updateUnconfirmed,
	}
)

// JSONIDGenerator sends incremental integers across a channel.  This
// is meant to provide a unique value for the JSON ID field for btcwallet
// messages.
func JSONIDGenerator(c chan uint64) {
	var n uint64
	for {
		c <- n
		n++
	}
}

var updateOnce sync.Once

// ListenAndUpdate opens a websocket connection to a btcwallet
// instance and initiates requests to fill the GUI with relevant
// information.
func ListenAndUpdate(certificates []byte, c chan error) {
	// Start each updater func in a goroutine.  Use a sync.Once to
	// ensure there are no duplicate updater functions running.
	updateOnce.Do(func() {
		for _, f := range updateFuncs {
			go f()
		}
	})

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certificates)
	tlsConfig := &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}

	// Create a websocket dialer that will be used to make the connection.
	// It is modified by the proxy setting below as needed.
	dialer := websocket.Dialer{TLSClientConfig: tlsConfig}

	if cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     cfg.Proxy,
			Username: cfg.ProxyUser,
			Password: cfg.ProxyPass,
		}
		dialer.NetDial = proxy.Dial
	}

	// btcwallet requires basic authorization, so we use a custom config
	// with the Authorization header set.
	login := cfg.Username + ":" + cfg.Password
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	requestHeader := make(http.Header)
	requestHeader.Add("Authorization", auth)

	// Connect to websocket.
	url := fmt.Sprintf("wss://%s/ws", cfg.RPCConnect)
	ws, _, err := dialer.Dial(url, requestHeader)
	if err != nil {
		log.Printf("[ERR] cannot create websocket config: %v", err)
		c <- ErrConnectionRefused
		return
	}
	c <- nil

	// Buffered channel for replies and notifications from btcwallet.
	replies := make(chan []byte, 100)

	go func() {
		for {
			// Receive message from wallet

			_, msg, err := ws.ReadMessage()
			if err != nil {
				close(replies)
				return
			}
			replies <- msg
		}
	}()

	for _, f := range walletReqFuncs {
		go f(ws)
	}

	for {
		select {
		case r, ok := <-replies:
			if !ok {
				// btcwallet connection lost.
				c <- ErrConnectionLost
				return
			}

			// Handle message here.
			go ProcessBtcwalletMessage(r)

		case <-triggers.newAddr:
			go cmdGetNewAddress(ws)

		case params := <-triggers.newWallet:
			go cmdCreateEncryptedWallet(ws, params)

		case <-triggers.lockWallet:
			go cmdWalletLock(ws)

		case params := <-triggers.unlockWallet:
			go cmdWalletPassphrase(ws, params)

		case pairs := <-triggers.sendTx:
			go cmdSendMany(ws, pairs)

		case fee := <-triggers.setTxFee:
			go cmdSetTxFee(ws, fee)
		}
	}
}

// ProcessBtcwalletMessage unmarshalls the JSON notification or
// reply received from btcwallet and decides how to handle it.
func ProcessBtcwalletMessage(b []byte) {
	// Idea: instead of reading btcwallet messages from just one
	// websocket connection, maybe use two so the same connection isn't
	// used for both notifications and responses?  Should make handling
	// must faster as unnecessary unmarshal attempts could be avoided.

	// Check for notifications first.
	if req, err := btcjson.ParseMarshaledCmd(b); err == nil {
		// btcwallet should not be sending Requests except for
		// notifications.  Check for a nil id.
		if req.Id() != nil {
			// Invalid response
			log.Printf("[WRN] btcwallet sent a non-notification JSON-RPC Request (Id: %v)",
				req.Id())
			return
		}

		// Message is a notification.  Check the method and dispatch
		// correct handler, or if no handler, log a warning.
		if ntfnHandler, ok := notificationHandlers[req.Method()]; ok {
			ntfnHandler(req)
		} else {
			// No handler; log warning.
			log.Printf("[WRN] unhandled notification with method %v",
				req.Method())
		}
		return
	}

	// b is not a Request notification, so it must be a Response.
	// Attempt to parse it as one and handle.
	var r btcjson.Reply
	if err := json.Unmarshal(b, &r); err != nil {
		log.Print("[WRN] Unable to unmarshal btcwallet message as notification or response")
		return
	}

	// Check for a valid ID.  btcgui only sends numbers as IDs, so
	// perform an appropiate type check.
	if r.Id == nil {
		// Responses with no IDs cannot be handled.
		log.Print("[WRN] Unable to process btcwallet response without ID")
		return
	}
	id, ok := (*r.Id).(float64)
	if !ok {
		log.Printf("[WRN] Unable to process btcwallet response with non-number ID %v",
			*r.Id)
		return
	}

	replyHandlers.Lock()
	defer replyHandlers.Unlock()
	if f, ok := replyHandlers.m[uint64(id)]; ok {
		delete(replyHandlers.m, uint64(id))
		f(r.Result, r.Error)
	} else {
		log.Print("[WRN] No handler for btcwallet response")
	}
}

type notificationHandler func(btcjson.Cmd)

var notificationHandlers = map[string]notificationHandler{
	btcws.BlockConnectedNtfnMethod:    handleBlockConnectedNtfn,
	btcws.BlockDisconnectedNtfnMethod: handleBlockDisconnectedNtfn,
	btcws.BtcdConnectedNtfnMethod:     handleBtcdConnectedNtfn,
	btcws.TxNtfnMethod:                handleTxNtfn,
	btcws.AccountBalanceNtfnMethod:    handleAccountBalanceNtfn,
	btcws.WalletLockStateNtfnMethod:   handleWalletLockStateNtfn,
}

// handleBlockConnectedNtfn handles btcd/btcwallet blockconnected
// notifications resulting from newly-connected blocks to the main
// blockchain.
func handleBlockConnectedNtfn(n btcjson.Cmd) {
	bcn, ok := n.(*btcws.BlockConnectedNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	updateChans.bcHeight <- bcn.Height
}

// handleBlockDisconnectedNtfn handles btcd/btcwallet blockdisconnected
// notifications resulting from blocks disconnected from the main chain.
//
// TODO(jrick): handle this by rolling back tx history and balances.
func handleBlockDisconnectedNtfn(n btcjson.Cmd) {
	bdn, ok := n.(*btcws.BlockDisconnectedNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	// TODO
	_ = bdn
}

// handleBtcdConnectedNtfn handles btcwallet btcdconnected notifications,
// updating the GUI with info about whether btcd is connected to btcwallet
// or not.
func handleBtcdConnectedNtfn(n btcjson.Cmd) {
	bcn, ok := n.(*btcws.BtcdConnectedNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	updateChans.btcdConnected <- bcn.Connected
}

// handleTxNtfn handles btcwallet newtx notifications by updating the GUI
// with details about a new tx to or from wallet addresses.
func handleTxNtfn(n btcjson.Cmd) {
	tn, ok := n.(*btcws.TxNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	// TODO(jrick): do proper filtering and display
	// tx details for all accounts.
	if tn.Account == "" {
		attr, err := NewTxAttributesFromJSON(tn.Details)
		if err != nil {
			log.Printf("[ERR] %v handler: bad details: %v",
				n.Method(), err)
			return
		}
		updateChans.prependOverviewTx <- attr
		updateChans.prependTx <- attr
	}
}

// handleAccountBalanceNtfn handles btcwallet accountbalance notifications by
// updating the GUI with either a new confirmed or unconfirmed balance.
func handleAccountBalanceNtfn(n btcjson.Cmd) {
	abn, ok := n.(*btcws.AccountBalanceNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	// TODO(jrick): do proper filtering and display all
	// account balances somewhere
	if abn.Account == "" {
		bal, _ := btcutil.NewAmount(abn.Balance)
		if abn.Confirmed {
			updateChans.balance <- bal
		} else {
			updateChans.unconfirmed <- bal
		}
	}

}

// handleWalletLockStateNtfn handles btcwallet walletlockstate notifications
// by updating the GUI with whether the wallet is locked or not, setting
// the sensitivity of various widgets for locking or unlocking the wallet.
func handleWalletLockStateNtfn(n btcjson.Cmd) {
	wlsn, ok := n.(*btcws.WalletLockStateNtfn)
	if !ok {
		log.Printf("[ERR] %v handler: unexpected type", n.Method())
		return
	}

	// TODO(jrick): do proper filtering and display all
	// account balances somewhere
	if wlsn.Account == "" {
		updateChans.lockState <- wlsn.Locked
	}
}

// cmdGetNewAddress requests a new wallet address.
//
// TODO(jrick): support non-default accounts
func cmdGetNewAddress(ws *websocket.Conn) {
	var err error
	defer func() {
		if err != nil {

		}
	}()

	n := <-NewJSONID
	msg, err := btcjson.CreateMessageWithId("getnewaddress", n, "")
	if err != nil {
		triggerReplies.newAddr <- err
		return
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		switch {
		case err == nil:
			if addr, ok := result.(string); ok {
				triggerReplies.newAddr <- addr
			}

		case err.Code == btcjson.ErrWalletKeypoolRanOut.Code:
			success := make(chan bool)
			glib.IdleAdd(func() {
				dialog, err := createUnlockDialog(unlockForKeypool, success)
				if err != nil {
					log.Print(err)
					success <- false
					return
				}
				dialog.Run()
			})
			if <-success {
				triggers.newAddr <- 1
			}

		default: // all other non-nil errors
			triggerReplies.newAddr <- errors.New(err.Message)
		}
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		triggerReplies.newAddr <- err
	}
}

// cmdCreateEncryptedWallet requests btcwallet to create a new wallet
// (or account), encrypted with the supplied passphrase.
func cmdCreateEncryptedWallet(ws *websocket.Conn, params *NewWalletParams) {
	n := <-NewJSONID
	cmd := btcws.NewCreateEncryptedWalletCmd(n, params.passphrase)
	msg, err := json.Marshal(cmd)
	if err != nil {
		triggerReplies.walletCreationErr <- err
		return
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			triggerReplies.walletCreationErr <- errors.New(err.Message)
		} else {
			triggerReplies.walletCreationErr <- nil

			// Request all wallet-related info again, now that the
			// default wallet is available.
			for _, f := range walletReqFuncs {
				go f(ws)
			}
		}
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		triggerReplies.walletCreationErr <- err
	}
}

// cmdGetAddressesByAccount requests all addresses for an account.
//
// TODO(jrick): support non-default accounts.
// TODO(jrick): stop throwing away errors.
func cmdGetAddressesByAccount(ws *websocket.Conn) {
	n := <-NewJSONID
	msg, err := btcjson.CreateMessageWithId("getaddressesbyaccount", n, "")
	if err != nil {
		updateChans.addrs <- []string{}
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if r, ok := result.([]interface{}); ok {
			addrs := []string{}
			for _, v := range r {
				addrs = append(addrs, v.(string))
			}
			updateChans.addrs <- addrs
		} else {
			if err.Code == btcjson.ErrWalletInvalidAccountName.Code {
				glib.IdleAdd(func() {
					if dialog, err := createNewWalletDialog(); err != nil {
						dialog.Run()
					}
				})
			}
			updateChans.addrs <- []string{}
		}
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		updateChans.addrs <- []string{}
	}
}

// cmdGetBalance requests the current balance (calculated with the default
// one confirmation).
func cmdGetBalance(ws *websocket.Conn) {
	n := <-NewJSONID
	cmd, err := btcjson.NewGetBalanceCmd(n)
	if err != nil {
		log.Printf("[ERR] cannot create getbalance command.")
		return
	}
	msg, _ := cmd.MarshalJSON()

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			log.Printf("[ERR] getbalance: %v", err)
			return
		}

		if result == nil {
			return
		}

		fbal, ok := result.(float64)
		if !ok {
			log.Printf("[ERR] getbalance reply is not a number.")
			return
		}
		bal, _ := btcutil.NewAmount(fbal)
		updateChans.balance <- bal
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		updateChans.addrs <- []string{}
	}
}

// cmdGetUnconfirmedBalance requests the current unconfirmed balance.
func cmdGetUnconfirmedBalance(ws *websocket.Conn) {
	n := <-NewJSONID
	cmd, err := btcws.NewGetUnconfirmedBalanceCmd(n)
	if err != nil {
		log.Printf("[ERR] cannot create getunconfirmedbalance command.")
		return
	}
	msg, _ := cmd.MarshalJSON()

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			log.Printf("[ERR] getunconfirmedbalance: %v", err)
			return
		}

		if result == nil {
			return
		}

		fbal, ok := result.(float64)
		if !ok {
			log.Printf("[ERR] getunconfirmedbalance reply is not a number.")
			return
		}
		bal, _ := btcutil.NewAmount(fbal)
		updateChans.unconfirmed <- bal
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		updateChans.addrs <- []string{}
	}
}

// cmdGetBlockCount request the height of the best chain.
func cmdGetBlockCount(ws *websocket.Conn) {
	n := <-NewJSONID
	cmd, err := btcjson.NewGetBlockCountCmd(n)
	if err != nil {
		log.Printf("[ERR] cannot create getblockcount command.")
		return
	}
	mcmd, _ := cmd.MarshalJSON()

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			log.Printf("[ERR] getblockcount: %v", err)
			return
		}

		if result == nil {
			return
		}

		fcount, ok := result.(float64)
		if !ok {
			log.Printf("[ERR] getblockcount result is not a number.")
			return
		}

		updateChans.bcHeight <- int32(fcount)
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, mcmd); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
	}
}

// cmdListAllTransactions requests all transactions for the default account.
//
// TODO(jrick): support non-default accounts.
func cmdListAllTransactions(ws *websocket.Conn) {
	n := <-NewJSONID
	cmd, err := btcws.NewListAllTransactionsCmd(n, "")
	if err != nil {
		log.Printf("[ERR] cannot create listalltransactions command.")
		return
	}
	mcmd, _ := cmd.MarshalJSON()

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			log.Printf("[ERR] listtransactions: %v", err)
			return
		}

		if result == nil {
			return
		}

		vr, ok := result.([]interface{})
		if !ok {
			log.Printf("[ERR] listalltransactions reply is not an array.")
			return
		}
		for i, r := range vr {
			m, ok := r.(map[string]interface{})
			if !ok {
				log.Print("[ERR] listalltransactions: reply is not an array of JSON objects.")
				return
			}

			txAttr, err := NewTxAttributesFromMap(m)
			if err != nil {
				log.Printf("[ERR] listalltransactions: %v", err)
				return
			}

			updateChans.appendTx <- txAttr

			if i < NOverviewTxs {
				updateChans.appendOverviewTx <- txAttr
			}
		}
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, mcmd); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
	}
}

// cmdWalletIsLocked requests the current lock state of the
// currently-opened wallet.
//
// TODO(jrick): stop throwing away errors.
func cmdWalletIsLocked(ws *websocket.Conn) {
	n := <-NewJSONID
	m := btcjson.Message{
		Jsonrpc: "1.0",
		Id:      n,
		Method:  "walletislocked",
		Params:  []interface{}{},
	}
	msg, err := json.Marshal(&m)
	if err != nil {
		return
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if r, ok := result.(bool); ok {
			updateChans.lockState <- r
		}
	}
	replyHandlers.Unlock()

	if err = ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		// TODO(jrick): what to send here?
	}
}

// cmdWalletLock locks the currently-opened wallet.  A reply handler
// is not set up because the GUI will be updated after a
// "btcwallet:newwalletlockstate" notification is sent.
func cmdWalletLock(ws *websocket.Conn) error {
	n := <-NewJSONID
	msg, err := btcjson.CreateMessageWithId("walletlock", n)
	if err != nil {
		return err
	}

	// We don't actually care about the reply and rely on the
	// notification to update widgets, but adding an empty handler
	// prevents warnings printed to logging output.
	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {}
	replyHandlers.Unlock()

	return ws.WriteMessage(websocket.TextMessage, msg)
}

// cmdWalletPassphrase requests wallet to store the encryption
// passphrase for the currently-opened wallet in memory for a given
// number of seconds.
func cmdWalletPassphrase(ws *websocket.Conn, params *UnlockParams) error {
	n := <-NewJSONID
	m := btcjson.Message{
		Jsonrpc: "1.0",
		Id:      n,
		Method:  "walletpassphrase",
		Params: []interface{}{
			params.passphrase,
			params.timeout,
		},
	}
	msg, _ := json.Marshal(&m)

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		triggerReplies.unlockSuccessful <- err == nil
	}
	replyHandlers.Unlock()

	return ws.WriteMessage(websocket.TextMessage, msg)
}

// cmdSendMany requests wallet to create a new transaction to one or
// more recipients.
//
// TODO(jrick): support non-default accounts
func cmdSendMany(ws *websocket.Conn, pairs map[string]float64) error {
	n := <-NewJSONID
	m := btcjson.Message{
		Jsonrpc: "1.0",
		Id:      n,
		Method:  "sendmany",
		Params: []interface{}{
			"",
			pairs,
		},
	}
	msg, err := json.Marshal(m)
	if err != nil {
		log.Print(err.Error())
		return err
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			triggerReplies.sendTx <- err
		} else {
			// success
			triggerReplies.sendTx <- nil
		}
	}
	replyHandlers.Unlock()

	return ws.WriteMessage(websocket.TextMessage, msg)
}

// cmdSetTxFee requests wallet to set the global transaction fee added
// to newly-created transactions and awarded to the block miner who
// includes the transaction.
func cmdSetTxFee(ws *websocket.Conn, fee float64) error {
	n := <-NewJSONID
	msg, err := btcjson.CreateMessageWithId("settxfee", n, fee)
	if err != nil {
		triggerReplies.setTxFeeErr <- err
		return err // TODO(jrick): this gets thrown away so just send via triggerReplies.
	}

	replyHandlers.Lock()
	replyHandlers.m[n] = func(result interface{}, err *btcjson.Error) {
		if err != nil {
			triggerReplies.setTxFeeErr <- err
		} else {
			// success
			triggerReplies.setTxFeeErr <- nil
		}
	}
	replyHandlers.Unlock()

	return ws.WriteMessage(websocket.TextMessage, msg)
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
	// Statusbar messages for various connection states.
	btcdd := "Disconnected from btcd"
	btcwc := "Established connection to btcwallet"
	btcwd := "Disconnected from btcwallet.  Attempting reconnect..."

	for {
		select {
		case conn := <-updateChans.btcwalletConnected:
			if conn {
				glib.IdleAdd(func() {
					//MenuBar.Settings.New.SetSensitive(true)
					//MenuBar.Settings.Encrypt.SetSensitive(true)
					MenuBar.Settings.TxFee.SetSensitive(true)
					// Lock/Unlock sensitivity is set by wallet notification.
					RecvCoins.NewAddrBtn.SetSensitive(true)
					StatusElems.Lab.SetText(btcwc)
					StatusElems.Pb.Hide()
				})
			} else {
				glib.IdleAdd(func() {
					//MenuBar.Settings.New.SetSensitive(false)
					//MenuBar.Settings.Encrypt.SetSensitive(false)
					MenuBar.Settings.Lock.SetSensitive(false)
					MenuBar.Settings.Unlock.SetSensitive(false)
					MenuBar.Settings.TxFee.SetSensitive(false)
					SendCoins.SendBtn.SetSensitive(false)
					RecvCoins.NewAddrBtn.SetSensitive(false)
					StatusElems.Lab.SetText(btcwd)
					StatusElems.Pb.Hide()
				})
			}
		case conn := <-updateChans.btcdConnected:
			if conn {
				glib.IdleAdd(func() {
					SendCoins.SendBtn.SetSensitive(true)
				})
			} else {
				glib.IdleAdd(func() {
					SendCoins.SendBtn.SetSensitive(false)
					StatusElems.Lab.SetText(btcdd)
					StatusElems.Pb.Hide()
				})
			}
		}
	}
}

// updateAddresses listens for new wallet addresses, updating the GUI when
// necessary.
func updateAddresses() {
	for {
		addrs := <-updateChans.addrs
		glib.IdleAdd(func() {
			RecvCoins.Store.Clear()
		})
		for i := range addrs {
			addr := addrs[i]
			glib.IdleAdd(func() {
				iter := RecvCoins.Store.Append()
				RecvCoins.Store.Set(iter, []int{1},
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
		balStr := balance.String()
		glib.IdleAdd(func() {
			Overview.Balance.SetMarkup("<b>" + balStr + "</b>")
			SendCoins.Balance.SetText("Balance: " + balStr)
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
		balStr := "<b>" + unconfirmed.String() + "</b>"
		glib.IdleAdd(func() {
			Overview.Unconfirmed.SetMarkup(balStr)
		})
	}
}

// updateLockState updates the application widgets due to a change in
// the currently-open wallet's lock state.
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

func updateTransactions() {
	for {
		select {
		case attr := <-updateChans.appendTx:
			glib.IdleAdd(func() {
				iter := txWidgets.store.Append()
				const layout = "01/02/2006"
				txWidgets.store.Set(iter, []int{0, 1, 2, 3},
					[]interface{}{attr.Date.Format(layout),
						attr.Direction.String(),
						attr.Address,
						attr.Amount.String()})
			})

		case attr := <-updateChans.appendOverviewTx:
			glib.IdleAdd(func() {
				txLabel, err := createTxLabel(attr)
				if err != nil {
					log.Printf("[ERR] cannot create tx label: %v\n", err)
					return
				}

				if len(Overview.TxList) == NOverviewTxs {
					first := Overview.TxList[0]
					copy(Overview.TxList, Overview.TxList[1:])
					Overview.TxList[NOverviewTxs-1] = txLabel
					Overview.Txs.Remove(first)
					first.Destroy()
				} else {
					Overview.TxList = append(Overview.TxList, txLabel)
				}

				Overview.Txs.Add(txLabel)

				txLabel.ShowAll()
			})

		case attr := <-updateChans.prependTx:
			glib.IdleAdd(func() {
				iter := txWidgets.store.Prepend()
				const layout = "01/02/2006"
				txWidgets.store.Set(iter, []int{0, 1, 2, 3},
					[]interface{}{attr.Date.Format(layout),
						attr.Direction.String(),
						attr.Address,
						attr.Amount.String()})
			})

		case attr := <-updateChans.prependOverviewTx:
			glib.IdleAdd(func() {
				txLabel, err := createTxLabel(attr)
				if err != nil {
					log.Printf("[ERR] cannot create tx label: %v\n", err)
					return
				}

				if len(Overview.TxList) == NOverviewTxs {
					last := Overview.TxList[NOverviewTxs-1]
					copy(Overview.TxList[1:], Overview.TxList)
					Overview.TxList[0] = txLabel
					Overview.Txs.Remove(last)
					last.Destroy()
				} else {
					Overview.TxList = append(Overview.TxList, txLabel)
				}

				Overview.Txs.InsertRow(0)
				Overview.Txs.Attach(txLabel, 0, 0, 1, 1)

				txLabel.ShowAll()
			})
		}
	}
}
