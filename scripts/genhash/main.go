package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	passwords := []struct{ label, pwd string }{
		{"administrador", "Admin@123"},
		{"mecanico", "Mecanico@123"},
		{"atendente", "Atende@123"},
	}
	for _, p := range passwords {
		h, _ := bcrypt.GenerateFromPassword([]byte(p.pwd), bcrypt.DefaultCost)
		fmt.Printf("-- %s / senha: %s\n%s\n\n", p.label, p.pwd, string(h))
	}
}
