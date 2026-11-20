// Cryptocompat is the Go half of scripts/crypto_compat.py; only public test
// vectors are used by that script. JSON on stdin avoids shell quoting issues.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"plainmark/internal/cryptoblock"
)

func main() {
	var request struct{ Mode, Text, Password string }
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
		panic(err)
	}
	var result string
	var err error
	switch request.Mode {
	case "encrypt":
		result, err = cryptoblock.Encrypt(request.Text, request.Password)
	case "decrypt":
		result, err = cryptoblock.Decrypt(request.Text, request.Password)
	default:
		err = fmt.Errorf("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
