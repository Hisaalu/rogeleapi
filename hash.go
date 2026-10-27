package main

import (
	"fmt"
	"github.com/Hisaalu/rogele/internal/auth"
)

func main() {
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		panic(err)
	}
	fmt.Println(hash)
}
