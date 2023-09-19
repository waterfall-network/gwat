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
func GetPassPhraseWithList(text string, confirmation bool, index int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if index < len(passwords) {
			return passwords[index]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	password := GetPassPhrase(text, confirmation)
	return password
}

// GetPassPhrase displays the given text(prompt) to the user and requests some textual
// data to be entered, but one which must not be echoed out into the terminal.
// The method returns the input provided by the user.
func GetPassPhrase(text string, confirmation bool) string {
	if text != "" {
		fmt.Println(text)
	}
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
func unlockAccount(ks *keystore.KeyStore, address string, pos int, passwords []string) (accounts.Account, string) {
	account, err := MakeAddress(ks, address)
	if err != nil {
		Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)

		password := GetPassPhraseWithList(prompt, false, pos, passwords)
		prompt1 := fmt.Sprintf("Unlocking account %s | Attempt %d/%d, | Password %s", address, trials+1, 3, passwords)
		log.Info(prompt1)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your password against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your password unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func makePasswordList(path string) []string {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		Fatalf("Failed to read password file: %v", err)
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
