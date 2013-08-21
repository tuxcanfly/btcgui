btcgui
======

btcgui is a graphical frontend to btcwallet and btcd written using gotk3.

This project is currently under active development is not production
ready yet.  Do not expect anything to work properly.  However, feel
free to poke through the source, especially if you are looking for an
example of a real, modern GUI application written using gotk3.

## Installation

btcgui can be installed with the go get command:

```bash
go get github.com/conformal/btcgui
```

## Running

To use btcgui, you must have btcd and btcwallet installed and running.
Assumming btcwallet is running on the default port 8332, btcgui can be
started without providing any command line arguments.

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

## TODO
- Connect btcwallet responses and notifications to update widgets
- Update widgets to reflect fact of btcwallet only supporting encrypted wallets
- Documentation
- Code cleanup
- Optimize
- Much much more.  Stay tuned.

## License

btcgui is licensed under the liberal ISC License.
