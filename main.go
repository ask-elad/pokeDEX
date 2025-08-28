package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Print("hello\n")
	words := cleanInput("hola amigo kaise ho thee ko")
	fmt.Println(words)
}

func cleanInput(text string) []string {
	cleaned := strings.TrimSpace(strings.ToLower(text))
	return strings.Fields(cleaned)
}
