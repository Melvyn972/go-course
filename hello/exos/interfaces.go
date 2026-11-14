package main

import "fmt"

// ─────────────────────────────────────────────────────────────
// Exercise: interfaces — "the playlist"
//
// You have been using interfaces all along without knowing it:
// fmt.Println accepts `any`, and when a value has a String()
// method, fmt uses it (that method is the fmt.Stringer interface).
//
// Once the TODOs below are done, uncomment playlistDemo()
// and call it from main().
// ─────────────────────────────────────────────────────────────

type Song struct {
	Title  string
	Artist string
	Secs   int
}

type Podcast struct {
	Title   string
	Episode int
	Secs    int
}

// TODO 1 — Give Song and Podcast a String() string method so that
// fmt.Println prints them nicely, e.g.:
//
//	🎵 "Numb" by Linkin Park (3:07)
//	🎙 "The Go Time Podcast" ep. 12 (45:10)
//
// Nothing to declare: having the method IS implementing fmt.Stringer.

func (s Song) String() string {
	return fmt.Sprintf("%q by %s (%s)", s.Title, s.Artist, mmss(s.Secs))
}

func (p Podcast) String() string {
	return fmt.Sprintf("%q ep. %d (%s)", p.Title, p.Episode, mmss(p.Secs))
}

// TODO 2 — Define your own interface:
//
//	type Audio interface {
//		Name() string
//		Duration() int // seconds
//	}
//
// then implement it for Song and Podcast (again: just write the methods).

type Audio interface {
	Name() string
	Duration() int
}

func (s Song) Name() string {
	return s.Title
}

func (s Song) Duration() int {
	return s.Secs
}

func (p Podcast) Name() string {
	return fmt.Sprintf("%s ep. %d", p.Title, p.Episode)
}

func (p Podcast) Duration() int {
	return p.Secs
}

// TODO 3 — Write:
//
//	func totalPlayTime(items []Audio) int
//
// that sums the Duration() of everything in the slice, whatever the
// concrete types are.

func totalPlayTime(items []Audio) int {
	total := 0
	for _, item := range items {
		total += item.Duration()
	}
	return total
}

// Expected output of playlistDemo():
//
//	🎵 "Numb" by Linkin Park (3:07)
//	🎙 "The Go Time Podcast" ep. 12 (45:10)
//	🎵 "Everlong" by Foo Fighters (4:10)
//	Total play time: 52:27

func playlistDemo() {
	playlist := []Audio{
		Song{Title: "Numb", Artist: "Linkin Park", Secs: 187},
		Podcast{Title: "The Go Time Podcast", Episode: 12, Secs: 2710},
		Song{Title: "Everlong", Artist: "Foo Fighters", Secs: 250},
	}
	for _, item := range playlist {
		fmt.Println(item)
	}
	fmt.Println("Total play time:", mmss(totalPlayTime(playlist)))
}

// mmss formats seconds as "m:ss", e.g. 187 -> "3:07".
func mmss(secs int) string {
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func main() {
	playlistDemo()
}
