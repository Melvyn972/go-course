package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
)

const (
	maxNumber = 100
	maxAttempts = 7
)

var scanner = bufio.NewScanner(os.Stdin)

// prompts the player and returns the parsed number
func readGuess() (int, error) {
	fmt.Print("Your guess: ")
	if !scanner.Scan() {
		fmt.Println("Bye!")
		os.Exit(0)
	}
	return strconv.Atoi(strings.TrimSpace(scanner.Text()))
}

// compare the secret to the player input
func compare(guess, secret int) int {
	switch {
	case guess < secret:
		return -1
	case guess > secret:
		return 1
	default:
		return 0
	}
}

// runs one game and return the attempts used and whether the player won
func play() (attempts int, won bool) {
	secret := rand.IntN(maxNumber) + 1

	for attempts < maxAttempts {
		guess, err := readGuess()
		if err != nil || guess < 1 || guess > maxNumber {
			fmt.Printf("Please enter a whole number between 1 and %d\n", maxNumber)
			continue
		}
		attempts++

		switch compare(guess, secret) {
		case -1:
			fmt.Println("Too low!")
		case 1:
			fmt.Println("Too high!")
		default:
			return attempts, true
		}
	}

	fmt.Printf("Out of attempts ! The secret number was %d !\n", secret)
	return attempts, false
}

func main() {
	defer fmt.Println("Thanks for playing !")
	fmt.Printf("I am thinking of a number between 1 and %d. You have %d attempts to find it.\n", maxNumber, maxAttempts)

	attempts, won := play()
	if won {
		fmt.Printf("Congratulations ! You have found the correct number in %d attempts !", attempts)
	}
}
