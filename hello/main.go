package main

import (
	"fmt"
	"maps"
)

func main() {
	m := make(map[string]int)
	m["k1"] = 3
	m["k2"] = 18
	fmt.Println("map: ", m)

	v1 := m["k1"]
	fmt.Println("v1: ", v1)
	
	v3 := m["k3"]
	fmt.Println("v3: ", v3)

	fmt.Println("len: ", len(m))

	delete(m, "k2")
	fmt.Println("mapdel: ", m)

	clear(m)
	fmt.Println("mapclr: ", m)

	_, prs := m["k2"]
	fmt.Println("prs: ", prs)

	n := map[string]int{"foo": 1, "bar": 2}
	fmt.Println("n: ", n)

	n2 := map[string]int{"foo": 1, "bar": 3}
	if !maps.Equal(n, n2) {
		fmt.Println("n != n2")
	}
}
