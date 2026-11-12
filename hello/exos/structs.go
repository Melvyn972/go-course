package main

type User struct {
	Name string
	Membership
}

type Membership struct {
	Type string
	MessageCharLimit int
}

func newUser(name string, memberShipType string) User {
	membership := Membership{Type: memberShipType}
	if memberShipType == "premium" {
		membership.MessageCharLimit = 1000
	} else {
		membership.MessageCharLimit = 100
	}
	return User{Name: name, Membership: membership}
}
