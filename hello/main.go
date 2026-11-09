package main

import "fmt"

func main() {
	for i := 0; i <= 3; i++ {
		fmt.Printf("%d", i)
	}

	for j := range 3 {
		fmt.Printf("%d", j)
	}
}
