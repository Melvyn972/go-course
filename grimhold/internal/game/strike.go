package game

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
)

// Action is the piece of a stat block combat reads.
type Action struct {
	Name        string
	Desc        string
	AttackBonus int
}

// Strike is the melee attack used for a riposte.
type Strike struct {
	Name  string
	Bonus int
	Parts []string
}

// Breath is the wyrmling cone, read from the action text.
type Breath struct {
	Name     string
	Dice     string
	DC       int
	Recharge int
}

var (
	toHitRe    = regexp.MustCompile(`(?i)\+\s*(\d+)\s+to hit`)
	parenRe    = regexp.MustCompile(`\(([^)]*)\)`)
	diceRe     = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)
	flatRe     = regexp.MustCompile(`(?i)(\d+)\s+(piercing|bludgeoning|slashing|necrotic|cold|fire|acid|poison|radiant|force|thunder|lightning|psychic)\b`)
	dcRe       = regexp.MustCompile(`(?i)DC\s+(\d+)`)
	diceHitRe  = regexp.MustCompile(`\((\d+d\d+)\)`)
	rechargeRe = regexp.MustCompile(`(?i)Recharge\s+(\d+)\s*[-–—]\s*(\d+)`)
)

// BestStrike picks the melee attack with the highest bonus.
// A tie goes to the Hit clause that lists more damage terms.
func BestStrike(actions []Action) (Strike, error) {
	var best *Strike
	for _, a := range actions {
		if !isMelee(a.Desc) {
			continue
		}
		parts, err := HitTerms(a.Desc)
		if err != nil {
			continue
		}
		bonus := a.AttackBonus
		if m := toHitRe.FindStringSubmatch(a.Desc); m != nil {
			n, _ := strconv.Atoi(m[1])
			bonus = n
		}
		s := Strike{Name: a.Name, Bonus: bonus, Parts: parts}
		if best == nil || s.Bonus > best.Bonus || (s.Bonus == best.Bonus && len(s.Parts) > len(best.Parts)) {
			cp := s
			best = &cp
		}
	}
	if best == nil {
		return Strike{}, fmt.Errorf("aucune attaque de mêlée lisible")
	}
	return *best, nil
}

func isMelee(desc string) bool {
	d := strings.ToLower(desc)
	return strings.Contains(d, "melee weapon attack") || strings.Contains(d, "melee spell attack")
}

// HitTerms reads the damage dice in a Hit clause.
// A "plus NdM" in that clause is included. Dice that follow a saving throw
// (poison, for example) are not: the sheet has no ability score to roll it.
func HitTerms(desc string) ([]string, error) {
	low := strings.ToLower(desc)
	i := strings.Index(low, "hit:")
	if i < 0 {
		return nil, fmt.Errorf("pas de clause Hit")
	}
	rest := desc[i+len("hit:"):]
	lower := strings.ToLower(rest)
	stop := len(rest)
	for _, phrase := range []string{"must make", "must succeed", "saving throw"} {
		if j := strings.Index(lower, phrase); j >= 0 && j < stop {
			stop = j
		}
	}
	window := rest[:stop]
	if parts := plusDice(window); len(parts) > 0 {
		return parts, nil
	}
	if m := flatRe.FindStringSubmatch(window); m != nil {
		return []string{m[1]}, nil
	}
	return nil, fmt.Errorf("dés de dégâts illisibles")
}

func plusDice(window string) []string {
	matches := parenRe.FindAllStringSubmatchIndex(window, -1)
	var parts []string
	prev := 0
	for _, loc := range matches {
		expr, ok := normDice(window[loc[2]:loc[3]])
		if !ok {
			continue
		}
		between := strings.ToLower(window[prev:loc[0]])
		if len(parts) == 0 {
			parts = append(parts, expr)
			prev = loc[1]
			continue
		}
		if strings.Contains(between, "plus") {
			parts = append(parts, expr)
			prev = loc[1]
			continue
		}
		break
	}
	return parts
}

func normDice(s string) (string, bool) {
	s = strings.ReplaceAll(strings.ToLower(s), " ", "")
	if diceRe.MatchString(s) {
		return s, true
	}
	return "", false
}

// ParseBreath reads DC, dice, and the recharge minimum from a breath action.
// It fails if those numbers are not in the text.
func ParseBreath(actions []Action) (Breath, error) {
	var found *Action
	for i := range actions {
		if strings.Contains(strings.ToLower(actions[i].Name), "breath") {
			found = &actions[i]
			break
		}
	}
	if found == nil {
		return Breath{}, fmt.Errorf("pas de souffle dans la fiche")
	}
	dcM := dcRe.FindStringSubmatch(found.Desc)
	diceM := diceHitRe.FindStringSubmatch(found.Desc)
	recM := rechargeRe.FindStringSubmatch(found.Name + " " + found.Desc)
	if dcM == nil || diceM == nil || recM == nil {
		return Breath{}, fmt.Errorf("souffle illisible")
	}
	dc, _ := strconv.Atoi(dcM[1])
	rec, _ := strconv.Atoi(recM[1])
	return Breath{Name: found.Name, Dice: diceM[1], DC: dc, Recharge: rec}, nil
}

func nomAttaque(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bite", "bites":
		return "Morsure"
	case "claw", "claws":
		return "Griffes"
	case "slam":
		return "Coup"
	case "pseudopod":
		return "Pseudopode"
	case "multiattack":
		return "Attaques multiples"
	case "cold breath":
		return "Souffle de froid"
	case "web":
		return "Toile"
	case "talons":
		return "Serres"
	case "beak":
		return "Bec"
	case "tentacle", "tentacles":
		return "Tentacules"
	case "ink cloud":
		return "Nuage d'encre"
	case "longsword":
		return "Épée longue"
	case "shortsword":
		return "Épée courte"
	case "scimitar":
		return "Cimeterre"
	case "strength drain":
		return "Drain de force"
	case "life drain":
		return "Drain de vie"
	case "smother":
		return "Étouffement"
	default:
		if strings.Contains(strings.ToLower(name), "breath") {
			return "Souffle"
		}
		return "Attaque"
	}
}

func describeAction(a Action) string {
	title := nomAttaque(a.Name)
	if isMelee(a.Desc) {
		if parts, err := HitTerms(a.Desc); err == nil {
			bonus := a.AttackBonus
			if m := toHitRe.FindStringSubmatch(a.Desc); m != nil {
				bonus, _ = strconv.Atoi(m[1])
			}
			return fmt.Sprintf("%s : +%d pour toucher, %s.\n", title, bonus, strings.Join(parts, " + "))
		}
	}
	if strings.Contains(strings.ToLower(a.Name), "breath") {
		if br, err := ParseBreath([]Action{a}); err == nil {
			return fmt.Sprintf("%s : %s, DD %d, recharge %d.\n", title, br.Dice, br.DC, br.Recharge)
		}
	}
	if title != "Attaque" {
		return title + ".\n"
	}
	return ""
}

// Roll evaluates one expression: "1d8", "2d6+3", or a flat "1".
func Roll(rng *rand.Rand, expr string) (int, error) {
	expr = strings.ReplaceAll(strings.ToLower(expr), " ", "")
	if expr == "" {
		return 0, fmt.Errorf("dés vides")
	}
	if !strings.Contains(expr, "d") {
		n, err := strconv.Atoi(expr)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("dés illisibles: %s", expr)
		}
		return n, nil
	}
	m := diceRe.FindStringSubmatch(expr)
	if m == nil {
		return 0, fmt.Errorf("dés illisibles: %s", expr)
	}
	n, _ := strconv.Atoi(m[1])
	sides, _ := strconv.Atoi(m[2])
	if n < 1 || n > 40 || sides < 1 || sides > 100 {
		return 0, fmt.Errorf("dés hors limite: %s", expr)
	}
	total := 0
	for i := 0; i < n; i++ {
		total += 1 + rng.IntN(sides)
	}
	if m[3] != "" {
		mod, err := strconv.Atoi(m[3])
		if err != nil {
			return 0, fmt.Errorf("dés illisibles: %s", expr)
		}
		total += mod
	}
	if total < 0 {
		total = 0
	}
	return total, nil
}

func rollSum(rng *rand.Rand, parts []string) (int, error) {
	total := 0
	for _, p := range parts {
		n, err := Roll(rng, p)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}
