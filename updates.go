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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/conformal/btcws"
	"github.com/conformal/go-socks"
	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"time"
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
		balance            chan float64
		btcdConnected      chan bool
		btcwalletConnected chan bool
		bcHeight           chan int64
		bcHeightRemote     chan int64
		lockState          chan bool
		unconfirmed        chan float64
		appendTx           chan *TxAttributes
		prependTx          chan *TxAttributes
		appendOverviewTx   chan *TxAttributes
		prependOverviewTx  chan *TxAttributes
	}{
		addrs:              make(chan []string),
		balance:            make(chan float64),
		btcdConnected:      make(chan bool),
		btcwalletConnected: make(chan bool),
		bcHeight:           make(chan int64),
		bcHeightRemote:     make(chan int64),
		lockState:          make(chan bool),
		unconfirmed:        make(chan float64),
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

	// Connect to websocket.
	remote := net.JoinHostPort("localhost", cfg.Port)
	url := fmt.Sprintf("wss://%s/frontend", remote)
	config, err := websocket.NewConfig(url, "https://localhost/")
	if err != nil {
		log.Printf("[ERR] cannot create websocket config: %v", err)
		c <- ErrConnectionRefused
		return
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certificates)
	config.TlsConfig = &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}

	// btcwallet requires basic authorization, so we use a custom config
	// with the Authorization header set.
	login := cfg.Username + ":" + cfg.Password
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	config.Header.Add("Authorization", auth)

	// Attempt to connect to running btcwallet instance. Bail if it fails.
	var ws *websocket.Conn
	var cerr error
	if cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     cfg.Proxy,
			Username: cfg.ProxyUser,
			Password: cfg.ProxyPass,
		}
		conn, err := proxy.Dial("tcp", remote)
		if err != nil {
			log.Printf("Error connecting to proxy: %v", err)
			c <- ErrConnectionRefused
			return
		}

		tlsConn := tls.Client(conn, config.TlsConfig)
		ws, cerr = websocket.NewClient(config, tlsConn)
	} else {
		ws, cerr = websocket.DialConfig(config)
	}
	if cerr != nil {
		log.Printf("[ERR] Cannot create websocket client: %v", cerr)
		c <- ErrConnectionRefused
		return
	}
	c <- nil

	// Buffered channel for replies and notifications from btcwallet.
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
			var rply btcjson.Reply
			if err := json.Unmarshal(r, &rply); err != nil {
				log.Printf("Unable to unmarshal JSON reply: %v",
					err)
				continue
			}

			if rply.Id == nil {
				log.Print("Invalid JSON ID")
				continue
			}
			id := *(rply.Id)
			switch id.(type) {
			case float64:
				// json.Unmarshal unmarshalls all numbers as
				// float64
				uintID := uint64(id.(float64))
				replyHandlers.Lock()
				f := replyHandlers.m[uintID]
				delete(replyHandlers.m, uintID)
				replyHandlers.Unlock()
				if f != nil {
					go f(rply.Result, rply.Error)
				}
			case string:
				// Handle btcwallet notification.
				go handleBtcwalletNtfn(id.(string),
					rply.Result)
			}
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

// handleBtcwalletNtfn processes notifications from btcwallet and
// btcd, triggering the GUI updates associated with the notification.
//
// TODO(jrick): pass a btcws.Notification to this func.
func handleBtcwalletNtfn(id string, result interface{}) {
	switch id {
	// Global notifications
	case "btcwallet:btcdconnected":
		if r, ok := result.(bool); ok {
			updateChans.btcdConnected <- r
		}

	case "btcwallet:newblockchainheight":
		if r, ok := result.(float64); ok {
			updateChans.bcHeight <- int64(r)
		}

	// Notifications per wallet (account)
	case btcws.TxNtfnId:
		if r, ok := result.(map[string]interface{}); ok {
			account, ok := r["account"].(string)
			if !ok {
				return
			}
			details, ok := r["details"].(map[string]interface{})
			if !ok {
				return
			}

			attr, err := parseTxDetails(details)
			if err != nil {
				return
			}

			// TODO(jrick): do proper filtering and display
			// tx details for all accounts.
			if account == "" {
				updateChans.prependOverviewTx <- attr
				updateChans.prependTx <- attr
			}
		}

	case "btcwallet:accountbalance":
		if r, ok := result.(map[string]interface{}); ok {
			account, ok := r["account"].(string)
			if !ok {
				return
			}
			balance, ok := r["notification"].(float64)
			if !ok {
				return
			}
			// TODO(jrick): do proper filtering and display all
			// account balances somewhere
			if account == "" {
				updateChans.balance <- balance
			}
		}

	case "btcwallet:accountbalanceunconfirmed":
		if r, ok := result.(map[string]interface{}); ok {
			account, ok := r["account"].(string)
			if !ok {
				return
			}
			balance, ok := r["notification"].(float64)
			if !ok {
				return
			}
			// TODO(jrick): do proper filtering and display all
			// account balances somewhere
			if account == "" {
				updateChans.unconfirmed <- balance
			}
		}

	case "btcwallet:newwalletlockstate":
		if m, ok := result.(map[string]interface{}); ok {
			// We only care about the default account right now.
			if m["account"].(string) == "" {
				updateChans.lockState <- m["notification"].(bool)
			}
		}

	default:
		log.Printf("Unhandled message with id '%s'\n", id)
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

	if err = websocket.Message.Send(ws, msg); err != nil {
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
	m := &btcjson.Message{
		Jsonrpc: "1.0",
		Id:      n,
		Method:  "createencryptedwallet",
		Params: []interface{}{
			params.name,
			params.desc,
			params.passphrase,
		},
	}
	msg, err := json.Marshal(m)
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

	if err = websocket.Message.Send(ws, msg); err != nil {
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

	if err = websocket.Message.Send(ws, msg); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
		updateChans.addrs <- []string{}
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

			txAttr, err := parseTxDetails(m)
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

	if err = websocket.Message.Send(ws, mcmd); err != nil {
		replyHandlers.Lock()
		delete(replyHandlers.m, n)
		replyHandlers.Unlock()
	}
}

func parseTxDetails(m map[string]interface{}) (*TxAttributes, error) {
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
	amount, err := btcjson.JSONToAmount(famount)
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

	if err := websocket.Message.Send(ws, msg); err != nil {
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
	msg, err := btcjson.CreateMessage("walletlock")
	if err != nil {
		return err
	}

	return websocket.Message.Send(ws, msg)
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

	return websocket.Message.Send(ws, msg)
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

	return websocket.Message.Send(ws, msg)
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
				var iter gtk.TreeIter
				RecvCoins.Store.Append(&iter)
				RecvCoins.Store.Set(&iter, []int{1},
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
				var iter gtk.TreeIter
				txWidgets.store.Append(&iter)
				const layout = "01/02/2006"
				txWidgets.store.Set(&iter, []int{0, 1, 2, 3},
					[]interface{}{attr.Date.Format(layout),
						attr.Direction.String(),
						attr.Address,
						amountStr(attr.Amount)})
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
				var iter gtk.TreeIter
				txWidgets.store.Prepend(&iter)
				const layout = "01/02/2006"
				txWidgets.store.Set(&iter, []int{0, 1, 2, 3},
					[]interface{}{attr.Date.Format(layout),
						attr.Direction.String(),
						attr.Address,
						amountStr(attr.Amount)})
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
