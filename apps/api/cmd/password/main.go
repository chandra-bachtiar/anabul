// cmd/password prints a bcrypt hash for APP_PASSWORD_HASH, usage:
// go run ./cmd/password "my-secret".
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if password == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/password <password>")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create hash:", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
	fmt.Fprintln(os.Stderr, "\nPaste the value above into APP_PASSWORD_HASH in .env")
}
