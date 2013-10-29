btcgui
======

btcgui is a graphical frontend to btcwallet and btcd written using
gotk3.

Full btcwallet installation instructions can be found
[here](https://github.com/conformal/btcwallet).

This project is currently under active development is not production
ready yet.  Because of this, support for connecting to a mainnet
btcwallet is currently disable, and testnet must be used instead.

Do not expect anything to work properly.  However, feel
free to poke through the source, especially if you are looking for an
example of a real, modern GUI application written using gotk3.

## Installation

### Windows - MSI Available

Install the btcd suite MSI available here:

https://github.com/conformal/btcd/releases

### Linux/BSD/POSIX - Build from Source

- Install Go according to the installation instructions here:
  http://golang.org/doc/install

- Install GTK.  As btcgui relies on
  [GoTK3](https://github.com/conformal/gotk3) for GTK binding support, a
  version compatible with GoTK3 is required.  If not using the latest
  supported version, an additional ```-tags gtk_#_#``` with your
  installed version must be specified with the ``go get`` command below
  when installing btcgui.  See the
  [GoTK3](https://github.com/conformal/gotk3) README for how build tags
  work.

- Run the following commands to obtain btcgui, all dependencies, and install it:
```bash
$ go get -u -v github.com/conformal/btcd/...
$ go get -u -v github.com/conformal/btcwallet/...
$ go get -u -v github.com/conformal/btcgui/...
```

- btcd and btcwallet will now be installed in either ```$GOROOT/bin``` or
  ```$GOPATH/bin``` depending on your configuration.  If you did not already
  add to your system path during the installation, we recommend you do so now.


## Updating

### Windows

Install a newer btcd suite MSI here:

https://github.com/conformal/btcd/releases

### Linux/BSD/POSIX - Build from Source

- Run the following commands to update btcwallet, all dependencies, and install it:

```bash
$ go get -u -v github.com/conformal/btcd/...
$ go get -u -v github.com/conformal/btcwallet/...
$ go get -u -v github.com/conformal/btcgui/...
```

Remember to specify the correct build tags when building btcgui.

## Getting Started

### Windows (Installed from MSI)

Open btcdsuite.bat from the Btcd Suite folder in the Start Menu.

### Linux/BSD/POSIX/Source

- Run the following commands to start btcd, btcwallet, and btcgui:

```bash
$ btcd --testnet -u rpcuser -P rpcpass
$ btcwallet -u rpcuser -P rpcpass
$ btcgui
```

## TODO
- Implement transaction lists
- Implement an address book
- Documentation
- Code cleanup
- Optimize
- Much much more.  Stay tuned.

## GPG Verification Key

All official release tags are signed by Conformal so users can ensure the code
has not been tampered with and is coming from Conformal.  To verify the
signature perform the following:

- Download the public key from the Conformal website at
  https://opensource.conformal.com/GIT-GPG-KEY-conformal.txt

- Import the public key into your GPG keyring:
  ```bash
  gpg --import GIT-GPG-KEY-conformal.txt
  ```

- Verify the release tag with the following command where `TAG_NAME` is a
  placeholder for the specific tag:
  ```bash
  git tag -v TAG_NAME
  ```

## License

btcgui is licensed under the liberal ISC License.
