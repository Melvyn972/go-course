package repl

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var useColor bool

var dangerRe = regexp.MustCompile(`([Dd]anger )(\d+)`)

func isTerm(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func houseColor(house string) string {
	switch house {
	case "gryffondor":
		return "\033[31m"
	case "serdaigle":
		return "\033[34m"
	case "poufsouffle":
		return "\033[33m"
	case "serpentard":
		return "\033[32m"
	default:
		return "\033[37m"
	}
}

func wandColor(id string) string {
	switch id {
	case "phenix":
		return "\033[31m"
	case "dragon":
		return "\033[91m"
	case "licorne":
		return "\033[97m"
	default:
		return "\033[37m"
	}
}

func spellColor(id string) string {
	switch id {
	case "burning-hands":
		return "\033[31m"
	case "shield":
		return "\033[36m"
	case "cure-wounds":
		return "\033[32m"
	case "sleep":
		return "\033[35m"
	case "magic-missile":
		return "\033[33m"
	default:
		return "\033[37m"
	}
}

func sceneColor(id string) string {
	switch id {
	case "black-lake", "chamber":
		return "\033[36m"
	case "forbidden-edge", "pres-au-lard":
		return "\033[32m"
	case "hospital-wing":
		return "\033[37m"
	case "trophy-room":
		return "\033[33m"
	case "iron-vault":
		return "\033[36m"
	default:
		return "\033[37m"
	}
}

func foeColor(slug string) string {
	switch slug {
	case "reef-shark", "giant-octopus", "shadow", "specter", "white-dragon-wyrmling":
		return "\033[36m"
	case "poisonous-snake", "giant-poisonous-snake", "swarm-of-poisonous-snakes", "giant-spider", "giant-wolf-spider":
		return "\033[32m"
	case "ghoul", "giant-rat", "swarm-of-rats", "wolf", "swarm-of-insects":
		return "\033[33m"
	case "owl", "flying-sword", "animated-armor", "rug-of-smothering":
		return "\033[37m"
	case "mimic":
		return "\033[31m"
	default:
		return "\033[37m"
	}
}

func dangerColor(n int) string {
	switch {
	case n <= 1:
		return "\033[2m"
	case n >= 4:
		return "\033[31m"
	default:
		return "\033[33m"
	}
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + "\033[0m"
}

func hpBar(hp, max int, color string) string {
	const n = 20
	filled := 0
	if max > 0 && hp > 0 {
		filled = hp * n / max
		if filled < 1 {
			filled = 1
		}
		if filled > n {
			filled = n
		}
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", n-filled)
	return paint(color, bar) + fmt.Sprintf(" %d/%d", hp, max)
}

func colorizeDanger(s string) string {
	if !useColor {
		return s
	}
	return dangerRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := dangerRe.FindStringSubmatch(m)
		n, _ := strconv.Atoi(parts[2])
		return parts[1] + paint(dangerColor(n), parts[2])
	})
}
