package game

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/VoluteTech/grimhold/internal/open5e"
	"github.com/VoluteTech/grimhold/internal/world"
)

// ErrNoSave is returned when save.json is not there.
var ErrNoSave = errors.New("aucune sauvegarde")

// Foe is the creature currently in the room. HP is the only mutable stat.
type Foe struct {
	Slug        string
	Name        string
	Type        string
	AC          int
	CR          string
	HP          int
	Actions     []Action
	Immunities  string
	BreathSpent bool
}

func (f *Foe) label() string { return Label(f.Slug, f.Name) }

// Game is one play session. Save writes it; death does not.
type Game struct {
	World *world.World
	API   *open5e.Client
	rng   *rand.Rand

	Loc        string
	Char       Character
	Foe        *Foe
	Rested     bool
	DragonDown bool
	Shield     bool
	Next       string
	Prev       string
	ready      bool
}

// New starts at the world's entrance. The sheet is filled by Start or Load.
func New(w *world.World, api *open5e.Client) *Game {
	return &Game{
		World: w,
		API:   api,
		rng:   rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0x9e3779b97f4a7c15)),
		Loc:   w.Start,
	}
}

// Ready reports whether a sheet exists.
func (g *Game) Ready() bool { return g.ready }

// Start creates the level-1 character in the Hall.
func (g *Game) Start(house, wand string, spells []string) {
	g.Char = NewCharacter(house, wand, spells)
	g.Loc = g.World.Start
	g.Foe = nil
	g.Rested = false
	g.Shield = false
	g.ready = true
}

func (g *Game) region() (*world.Region, error) {
	r, ok := g.World.Get(g.Loc)
	if !ok {
		return nil, fmt.Errorf("lieu inconnu: %s", g.Loc)
	}
	return r, nil
}

// CanAmbush is true in a dangerous room with nobody already there.
func (g *Game) CanAmbush() bool {
	if g.Foe != nil {
		return false
	}
	r, err := g.region()
	if err != nil {
		return false
	}
	return r.Danger >= 2
}

// Ambush pulls one foe from the current room. One roll, not the explore rule.
func (g *Game) Ambush(ctx context.Context) (string, error) {
	if !g.CanAmbush() {
		return "", nil
	}
	r, err := g.region()
	if err != nil {
		return "", err
	}
	slug := r.Foes[g.rng.IntN(len(r.Foes))]
	m, err := g.API.Monster(ctx, slug)
	if err != nil {
		return "", err
	}
	g.setFoe(m)
	return "Le tronc crache une étincelle.\n" + g.announce(""), nil
}

// Explore rolls Danger times and keeps the highest challenge rating.
// A tie keeps the later roll.
func (g *Game) Explore(ctx context.Context) (string, error) {
	if g.Foe != nil {
		return "", errors.New("Un monstre est déjà là")
	}
	r, err := g.region()
	if err != nil {
		return "", err
	}
	var best *open5e.Monster
	var bestCR float64
	for i := 0; i < r.Danger; i++ {
		slug := r.Foes[g.rng.IntN(len(r.Foes))]
		m, err := g.API.Monster(ctx, slug)
		if err != nil {
			return "", err
		}
		cr := open5e.CRValue(string(m.ChallengeRating))
		if best == nil || cr >= bestCR {
			cp := m
			best = &cp
			bestCR = cr
		}
	}
	g.setFoe(*best)
	return g.announce("Devant toi : "), nil
}

func foeFrom(m open5e.Monster) *Foe {
	acts := make([]Action, len(m.Actions))
	for i, a := range m.Actions {
		acts[i] = Action{Name: a.Name, Desc: a.Desc, AttackBonus: a.AttackBonus}
	}
	return &Foe{
		Slug:       m.Slug,
		Name:       m.Name,
		Type:       m.Type,
		AC:         int(m.ArmorClass),
		CR:         string(m.ChallengeRating),
		HP:         m.HitPoints,
		Actions:    acts,
		Immunities: m.ConditionImmunities.String(),
	}
}

func (g *Game) setFoe(m open5e.Monster) {
	g.Foe = foeFrom(m)
	g.Shield = false
}

func (g *Game) announce(lead string) string {
	s := lead + g.Foe.label() + "."
	if g.Char.House == "serpentard" {
		s += fmt.Sprintf(" CA %d, FP %s.", g.Foe.AC, g.Foe.CR)
	}
	return s + "\n"
}

// Travel moves along an exit. The vault refuses anyone under level 4.
// Entering the chamber rolls the skull once.
func (g *Game) Travel(q string) (string, error) {
	if g.Foe != nil {
		return "", errors.New("Un monstre bloque le passage")
	}
	dest, err := g.World.Find(q)
	if err != nil {
		return "", err
	}
	cur, err := g.region()
	if err != nil {
		return "", err
	}
	if dest.ID == cur.ID {
		return "", errors.New("Tu es déjà ici")
	}
	if !cur.LeadsTo(dest.ID) {
		return "", fmt.Errorf("Aucun passage vers %s", dest.Name)
	}
	if dest.ID == regionVault && g.Char.Level < vaultLevel {
		return "", errors.New("La porte de fer est trop étroite. Il te faut le niveau 4")
	}
	g.Loc = dest.ID
	g.Rested = false
	g.Shield = false
	var b strings.Builder
	fmt.Fprintf(&b, "%s.\n%s\n", dest.Name, dest.Description)
	if dest.ID == regionChamber {
		b.WriteString(g.gaze())
	}
	return b.String(), nil
}

func (g *Game) gaze() string {
	roll := 1 + g.rng.IntN(20)
	if roll < gazeDC {
		g.Char.Gaze = true
		return fmt.Sprintf("Le crâne est dans l'axe de l'échelle (%d). -2 à l'attaque jusqu'au prochain repos.\n", roll)
	}
	g.Char.Gaze = false
	return fmt.Sprintf("Le crâne est de côté (%d).\n", roll)
}

// Look describes the room, its danger, and its exits.
func (g *Game) Look() (string, error) {
	r, err := g.region()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n", r.Name, r.Description)
	fmt.Fprintf(&b, "Danger %d, %s.\n", r.Danger, dangerHint(r.Danger))
	fmt.Fprintf(&b, "Sorties : %s\n", g.exitList(r))
	if r.ID == regionVault {
		b.WriteString("Entrée : niveau 4.\n")
	}
	if g.Foe != nil {
		b.WriteString(g.announce("Devant toi : "))
	}
	return b.String(), nil
}

// Map lists every region. The level gate does not hide the vault.
func (g *Game) Map() string {
	var b strings.Builder
	for i := range g.World.Regions {
		r := &g.World.Regions[i]
		here := ""
		if r.ID == g.Loc {
			here = " ← ici"
		}
		fmt.Fprintf(&b, "%s — danger %d, %s%s\n", r.Name, r.Danger, dangerHint(r.Danger), here)
		fmt.Fprintf(&b, "  sorties : %s\n", g.exitList(r))
		if r.ID == regionVault {
			b.WriteString("  entrée : niveau 4\n")
		}
	}
	return b.String()
}

// DangerText is the short French line for a room's danger.
func DangerText(d int) string { return dangerHint(d) }

func dangerHint(d int) string {
	switch {
	case d <= 1:
		return "un seul tirage"
	case d >= 4:
		return "plusieurs tirages, un seul monstre"
	default:
		return "plusieurs tirages, le pire reste"
	}
}

// Exit is one way out of the current room.
type Exit struct {
	ID   string
	Name string
}

// Exits lists the current room's exits, in world order.
func (g *Game) Exits() ([]Exit, error) {
	r, err := g.region()
	if err != nil {
		return nil, err
	}
	out := make([]Exit, len(r.Exits))
	for i, id := range r.Exits {
		name := id
		if dest, ok := g.World.Get(id); ok {
			name = dest.Name
		}
		out[i] = Exit{ID: id, Name: name}
	}
	return out, nil
}

func (g *Game) exitList(r *world.Region) string {
	if len(r.Exits) == 0 {
		return "aucune"
	}
	parts := make([]string, len(r.Exits))
	for i, id := range r.Exits {
		if dest, ok := g.World.Get(id); ok {
			parts[i] = dest.Name
		} else {
			parts[i] = id
		}
	}
	return strings.Join(parts, ", ")
}

// Rest heals, returns spells and patronum, and clears the gaze.
// Once per entry. Leaving and coming back arms it again.
func (g *Game) Rest() (string, error) {
	if g.Foe != nil {
		return "", errors.New("Un monstre est déjà là")
	}
	if g.Rested {
		return "", errors.New("Tu t'es déjà reposé ici")
	}
	g.Char.recover()
	g.Rested = true
	g.Shield = false
	return "PV au max. Sorts, patronum et soin rendus.\n", nil
}

// Sheet is the character screen, without the HP bar (the prompt draws that).
func (g *Game) Sheet() string {
	c := g.Char
	var b strings.Builder
	if r, err := g.region(); err == nil {
		fmt.Fprintf(&b, "Lieu      %s\n", r.Name)
	}
	fmt.Fprintf(&b, "Maison    %s\n", houseLabel[c.House])
	fmt.Fprintf(&b, "Baguette  %s (%s)\n", wandLabel[c.Wand], wandDice[c.Wand])
	if c.Wand == "licorne" {
		b.WriteString("          Episkey +2\n")
	}
	next := c.Level * c.Level * 50
	fmt.Fprintf(&b, "Niveau    %d\n", c.Level)
	fmt.Fprintf(&b, "PX        %d / %d\n", c.XP, next)
	fmt.Fprintf(&b, "PV        %d / %d\n", c.HP, c.MaxHP)
	fmt.Fprintf(&b, "CA        %d\n", playerAC)
	fmt.Fprintf(&b, "Attaque   %+d\n", c.AttackBonus())
	fmt.Fprintf(&b, "Fuite     +%d contre %d\n", c.FleeBonus(), fleeDC)
	fmt.Fprintf(&b, "Galions   %d\n", c.Gold)
	if c.Gaze {
		b.WriteString("Regard    -2\n")
	}
	b.WriteString("Sorts\n")
	for _, s := range c.Spells {
		state := "prêt"
		if c.SpentSpell(s) {
			state = "dépensé"
		}
		fmt.Fprintf(&b, "  %s  %s\n", spellLabel[s], state)
	}
	if c.SoinSpent {
		b.WriteString("Soin      dépensé\n")
	} else {
		b.WriteString("Soin      prêt\n")
	}
	if c.PatronumSpent {
		b.WriteString("Patronum  dépensé\n")
	} else {
		b.WriteString("Patronum  prêt\n")
	}
	if len(c.Kills) > 0 {
		b.WriteString("Tués\n")
		for _, slug := range sortedKills(c) {
			fmt.Fprintf(&b, "  %s × %d\n", killName(c, slug), c.Kills[slug])
		}
	}
	if g.DragonDown {
		b.WriteString("Le dragon du caveau est à terre.\n")
	}
	return b.String()
}

// Journal lists outcomes in the order they happened.
func (g *Game) Journal() string {
	if len(g.Char.Journal) == 0 {
		return "Rien de noté.\n"
	}
	var b strings.Builder
	for _, e := range g.Char.Journal {
		fmt.Fprintf(&b, "%s — %s\n", e.Name, e.Outcome)
	}
	return b.String()
}

// Inspect loads a sheet by slug, English name, or one of the three aliases.
func (g *Game) Inspect(ctx context.Context, q string) (string, error) {
	m, err := g.API.Monster(ctx, resolveSlug(q))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", Label(m.Slug, m.Name))
	fmt.Fprintf(&b, "Type %s, CA %d, PV %d, FP %s\n", frenchType(m.Type), int(m.ArmorClass), m.HitPoints, m.ChallengeRating)
	if s := frenchImmunities(m.ConditionImmunities.String()); s != "" {
		fmt.Fprintf(&b, "Immunités %s\n", s)
	}
	for _, a := range m.Actions {
		if line := describeAction(Action{Name: a.Name, Desc: a.Desc, AttackBonus: a.AttackBonus}); line != "" {
			b.WriteString(line)
		}
	}
	return b.String(), nil
}

// CanonSlug is the Open5e slug for a French name, an alias, or a slug.
func CanonSlug(q string) string { return resolveSlug(q) }

func resolveSlug(q string) string {
	key := world.Fold(q)
	if s, ok := nameToSlug[key]; ok {
		return s
	}
	return strings.ReplaceAll(key, " ", "-")
}

func frenchType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "aberration":
		return "aberration"
	case "beast":
		return "bête"
	case "celestial":
		return "céleste"
	case "construct":
		return "construction"
	case "dragon":
		return "dragon"
	case "elemental":
		return "élémentaire"
	case "fey":
		return "fée"
	case "fiend":
		return "fiélon"
	case "giant":
		return "géant"
	case "humanoid":
		return "humanoïde"
	case "monstrosity":
		return "monstruosité"
	case "ooze":
		return "limon"
	case "plant":
		return "plante"
	case "undead":
		return "mort-vivant"
	default:
		return "créature"
	}
}

func frenchImmunities(s string) string {
	known := map[string]string{
		"blinded":       "aveuglé",
		"charmed":       "charmé",
		"deafened":      "assourdi",
		"exhaustion":    "épuisement",
		"frightened":    "effrayé",
		"grappled":      "agrippé",
		"incapacitated": "neutralisé",
		"invisible":     "invisible",
		"paralyzed":     "paralysé",
		"petrified":     "pétrifié",
		"poisoned":      "empoisonné",
		"prone":         "à terre",
		"restrained":    "entravé",
		"stunned":       "étourdi",
		"unconscious":   "inconscient",
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' })
	var out []string
	for _, p := range parts {
		w := strings.ToLower(strings.TrimSpace(p))
		if w == "" {
			continue
		}
		if fr, ok := known[w]; ok {
			out = append(out, fr)
		}
	}
	return strings.Join(out, ", ")
}

// Bestiary follows the API's next and previous URLs.
func (g *Game) Bestiary(ctx context.Context, which string) (string, error) {
	var u string
	switch which {
	case "":
	case "next":
		if g.Next == "" {
			return "", errors.New("Pas de page suivante")
		}
		u = g.Next
	case "prev":
		if g.Prev == "" {
			return "", errors.New("Pas de page précédente")
		}
		u = g.Prev
	default:
		return "", errors.New("usage : bestiaire [suivant|précédent]")
	}
	p, err := g.API.Page(ctx, u)
	if err != nil {
		return "", err
	}
	g.Next, g.Prev = "", ""
	if p.Next != nil {
		g.Next = *p.Next
	}
	if p.Previous != nil {
		g.Prev = *p.Previous
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d fiches\n", p.Count)
	for _, m := range p.Results {
		fmt.Fprintf(&b, "%s  FP %s\n", Label(m.Slug, m.Name), m.ChallengeRating)
	}
	if g.Next != "" {
		b.WriteString("bestiaire suivant\n")
	}
	if g.Prev != "" {
		b.WriteString("bestiaire précédent\n")
	}
	return b.String(), nil
}
