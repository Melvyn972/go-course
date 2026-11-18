package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/VoluteTech/grimhold/internal/world"
)

const (
	playerAC      = 12
	fleeDC        = 12
	gazeDC        = 12
	vaultLevel    = 4
	regionVault   = "iron-vault"
	regionChamber = "chamber"
	dragonSlug    = "white-dragon-wyrmling"
)

var houseLabel = map[string]string{
	"gryffondor":  "Gryffondor",
	"serdaigle":   "Serdaigle",
	"poufsouffle": "Poufsouffle",
	"serpentard":  "Serpentard",
}

var houseAttack = map[string]int{"gryffondor": 1}
var houseFlee = map[string]int{"serdaigle": 2}
var houseHP = map[string]int{
	"gryffondor":  0,
	"serdaigle":   0,
	"poufsouffle": 2,
	"serpentard":  0,
}

var wandLabel = map[string]string{
	"phenix":  "Phénix",
	"dragon":  "Dragon",
	"licorne": "Licorne",
}

var wandDice = map[string]string{
	"phenix":  "1d8",
	"dragon":  "1d6",
	"licorne": "1d4",
}

var wandAttack = map[string]int{"dragon": 1}

var spellOrder = []string{"burning-hands", "shield", "cure-wounds", "sleep", "magic-missile"}

var spellLabel = map[string]string{
	"burning-hands": "Incendio",
	"shield":        "Protego",
	"cure-wounds":   "Episkey",
	"sleep":         "Sommeil",
	"magic-missile": "Projectiles magiques",
}

var creatureName = map[string]string{
	"giant-rat":                 "rat géant",
	"swarm-of-rats":             "nuée de rats",
	"owl":                       "hibou",
	"reef-shark":                "requin de récif",
	"poisonous-snake":           "serpent venimeux",
	"giant-poisonous-snake":     "serpent venimeux géant",
	"swarm-of-poisonous-snakes": "nuée de serpents venimeux",
	"wolf":                      "loup",
	"giant-wolf-spider":         "araignée-loup géante",
	"swarm-of-insects":          "nuée d'insectes",
	"shadow":                    "ombre",
	"specter":                   "spectre",
	"ghoul":                     "goule",
	"flying-sword":              "épée volante",
	"animated-armor":            "armure animée",
	"mimic":                     "mimique",
	"rug-of-smothering":         "tapis étouffeur",
}

var nameToSlug = map[string]string{}

func init() {
	for slug, fr := range creatureName {
		nameToSlug[world.Fold(fr)] = slug
	}
	for slug, a := range aliases {
		nameToSlug[world.Fold(a)] = slug
	}
	nameToSlug["chouette"] = "owl"
}

var aliases = map[string]string{
	"giant-octopus":         "calmar géant",
	"giant-spider":          "acromantula",
	"white-dragon-wyrmling": "ironbelly jeune",
}

// ValidHouse reports whether s is a creation choice.
func ValidHouse(s string) bool { _, ok := houseLabel[s]; return ok }

// ValidWand reports whether s is a creation choice.
func ValidWand(s string) bool { _, ok := wandDice[s]; return ok }

// ValidSpell reports whether s is one of the five castable slugs.
func ValidSpell(s string) bool { _, ok := spellLabel[s]; return ok }

// SpellList is the creation prompt, in slug order.
func SpellList() string { return strings.Join(spellOrder, ", ") }

// HouseTitle, WandTitle, and SpellTitle are the names shown in the menus.
func HouseTitle(s string) string {
	if t, ok := houseLabel[s]; ok {
		return t
	}
	return s
}

// WandTitle is the name shown for a wand core.
func WandTitle(s string) string {
	if t, ok := wandLabel[s]; ok {
		return t
	}
	return s
}

// SpellTitle is the name shown for a spell slug.
func SpellTitle(s string) string {
	if t, ok := spellLabel[s]; ok {
		return t
	}
	return s
}

// SpellOrder is the five castable slugs, in menu order.
func SpellOrder() []string { return append([]string(nil), spellOrder...) }

// Label is the name shown to the player: a Harry Potter term, else a French name.
func Label(slug, name string) string {
	if a, ok := aliases[slug]; ok {
		return a
	}
	if fr, ok := creatureName[slug]; ok {
		return fr
	}
	return name
}

// JournalEntry is one line of the slain, slept, or driven-off log.
type JournalEntry struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Outcome string `json:"outcome"`
}

// Character is the player sheet that save.json stores.
type Character struct {
	House         string         `json:"house"`
	Wand          string         `json:"wand"`
	Level         int            `json:"level"`
	XP            int            `json:"xp"`
	HP            int            `json:"hp"`
	MaxHP         int            `json:"maxHP"`
	Gold          int            `json:"gold"`
	Spells        []string       `json:"spells"`
	Spent         []string       `json:"spent"`
	PatronumSpent bool           `json:"patronumSpent"`
	SoinSpent     bool           `json:"soinSpent"`
	Gaze          bool           `json:"gaze"`
	Kills         map[string]int `json:"kills"`
	Journal       []JournalEntry `json:"journal"`
}

// NewCharacter builds a level-1 sheet. HP includes the house bonus.
func NewCharacter(house, wand string, spells []string) Character {
	c := Character{
		House:  house,
		Wand:   wand,
		Level:  1,
		Spells: append([]string(nil), spells...),
		Kills:  map[string]int{},
	}
	c.MaxHP = c.maxForLevel()
	c.HP = c.MaxHP
	return c
}

func (c Character) maxForLevel() int {
	return 10 + 4*(c.Level-1) + houseHP[c.House]
}

func (c *Character) sync() {
	c.MaxHP = c.maxForLevel()
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	if c.HP < 1 {
		c.HP = 1
	}
	if c.Kills == nil {
		c.Kills = map[string]int{}
	}
}

// AttackBonus is 2 + (level-1) + house + wand, minus 2 while the skull's gaze holds.
func (c Character) AttackBonus() int {
	b := 2 + (c.Level - 1) + houseAttack[c.House] + wandAttack[c.Wand]
	if c.Gaze {
		b -= 2
	}
	return b
}

// FleeBonus is the flat bonus on the d20 against fleeDC.
func (c Character) FleeBonus() int { return 2 + houseFlee[c.House] }

// Knows reports whether slug was picked at creation.
func (c Character) Knows(slug string) bool {
	for _, s := range c.Spells {
		if s == slug {
			return true
		}
	}
	return false
}

// SpentSpell reports whether slug was cast since the last rest.
func (c Character) SpentSpell(slug string) bool {
	for _, s := range c.Spent {
		if s == slug {
			return true
		}
	}
	return false
}

func (c *Character) spend(slug string) {
	if !c.SpentSpell(slug) {
		c.Spent = append(c.Spent, slug)
	}
}

func (c *Character) recover() {
	c.HP = c.MaxHP
	c.Spent = nil
	c.PatronumSpent = false
	c.SoinSpent = false
	c.Gaze = false
}

func (c *Character) gainXP(n int) []string {
	c.XP += n
	var lines []string
	for c.Level < 20 && c.XP >= c.Level*c.Level*50 {
		c.Level++
		c.MaxHP += 4
		c.HP += 4
		lines = append(lines, fmt.Sprintf("Niveau %d. PV max %d. Attaque %+d.", c.Level, c.MaxHP, c.AttackBonus()))
	}
	return lines
}

func (c *Character) note(slug, name, outcome string) {
	c.Journal = append(c.Journal, JournalEntry{Slug: slug, Name: name, Outcome: outcome})
}

func (c Character) validate() error {
	if _, ok := houseLabel[c.House]; !ok {
		return fmt.Errorf("maison inconnue dans la sauvegarde")
	}
	if _, ok := wandDice[c.Wand]; !ok {
		return fmt.Errorf("baguette inconnue dans la sauvegarde")
	}
	if c.Level < 1 {
		return fmt.Errorf("niveau invalide")
	}
	if len(c.Spells) != 2 || c.Spells[0] == c.Spells[1] || !ValidSpell(c.Spells[0]) || !ValidSpell(c.Spells[1]) {
		return fmt.Errorf("sorts invalides")
	}
	return nil
}

func killName(c Character, slug string) string {
	for i := len(c.Journal) - 1; i >= 0; i-- {
		if c.Journal[i].Slug == slug {
			return c.Journal[i].Name
		}
	}
	return slug
}

func sortedKills(c Character) []string {
	keys := make([]string, 0, len(c.Kills))
	for k := range c.Kills {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
