// cmd/password membuat hash bcrypt untuk APP_PASSWORD_HASH.
//
// Contoh:
//
//	go run ./cmd/password "rahasia-ku"
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
		fmt.Fprintln(os.Stderr, "pakai: go run ./cmd/password <password>")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal membuat hash:", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
	fmt.Fprintln(os.Stderr, "\nTempel nilai di atas ke APP_PASSWORD_HASH di .env")
}
