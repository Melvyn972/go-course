package main

import "fmt"

func main() {
	const name = "Saul Goodman"
	const openRate = 30.54

	msg := fmt.Sprintf("Hi %s, your open rate is %.2f percent\n", name, openRate)

	fmt.Print(msg)
}
