package creator

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts/keystore"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/console/prompt"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(ks *keystore.KeyStore, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	log.Warn("-------------------------------------------------------------------")
	log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	log.Warn("This functionality is deprecated and will be removed in the future!")
	log.Warn("Please use explicit addresses! (can search via `geth account list`)")
	log.Warn("-------------------------------------------------------------------")

	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}

// GetPassPhraseWithList retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func getPassPhraseWithList(confirmation bool, index int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if index < len(passwords) {
			return passwords[index]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	password := getPassPhrase(confirmation)
	return password
}

// GetPassPhrase displays the given text(prompt) to the user and requests some textual
// data to be entered, but one which must not be echoed out into the terminal.
// The method returns the input provided by the user.
func getPassPhrase(confirmation bool) string {
	password, err := prompt.Stdin.PromptPassword("Password: ")
	if err != nil {
		Fatalf("Failed to read password: %v", err)
	}
	if confirmation {
		confirm, err := prompt.Stdin.PromptPassword("Repeat password: ")
		if err != nil {
			Fatalf("Failed to read password confirmation: %v", err)
		}
		if password != confirm {
			Fatalf("Passwords do not match")
		}
	}
	return password
}

// findAccountPosition finds the position of a target account in a list of accounts.
func findAccountPosition(accounts []accounts.Account, targetAddress string) int {
	for index, account := range accounts {
		if account.Address.String() == targetAddress {
			return index
		}
	}
	return -1 // account not found
}

// tries unlocking the specified account a few times.
func unlockAccount(ks *keystore.KeyStore, address string, pos int, passwords []string) error {
	account, err := MakeAddress(ks, address)
	if err != nil {
		log.Error("Could not list accounts", "error", err)
		return err
	}

	for trials := 0; trials < 3; trials++ {
		password := getPassPhraseWithList(false, pos, passwords)
		err = ks.Unlock(account, password)
		if err != nil {
			log.Warn("Failed to unlock account", "account", address, "error", err)
			return err
		}
	}

	log.Info("Unlocked account", "address", account.Address.Hex())

	return nil
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func makePasswordList(path string) []string {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("Failed to read password file", "error", err)
		return nil
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

// fetchKeystore retrieves the encrypted keystore from the account manager.
func fetchKeystore(am *accounts.Manager) (*keystore.KeyStore, error) {
	if ks := am.Backends(keystore.KeyStoreType); len(ks) > 0 {
		return ks[0].(*keystore.KeyStore), nil
	}
	return nil, errors.New("local keystore not used")
}
