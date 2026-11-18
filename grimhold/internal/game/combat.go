package game

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/VoluteTech/grimhold/internal/open5e"
)

func xpFor(cr string) int {
	v := open5e.CRValue(cr)
	switch {
	case v <= 0:
		return 10
	case v <= 0.125:
		return 15
	case v <= 0.25:
		return 25
	case v <= 0.5:
		return 50
	case v <= 1:
		return 100
	default:
		return 200
	}
}

func (g *Game) prepare() (Strike, *Breath, error) {
	s, err := BestStrike(g.Foe.Actions)
	if err != nil {
		return Strike{}, nil, fmt.Errorf("attaque de %s : %w", g.Foe.label(), err)
	}
	if g.Foe.Slug != dragonSlug {
		return s, nil, nil
	}
	br, err := ParseBreath(g.Foe.Actions)
	if err != nil {
		return Strike{}, nil, fmt.Errorf("souffle de %s : %w", g.Foe.label(), err)
	}
	return s, &br, nil
}

// Fight is one exchange: the player swings, then the foe ripostes.
func (g *Game) Fight() (string, error) {
	if g.Foe == nil {
		return "", errors.New("Rien en face")
	}
	strike, breath, err := g.prepare()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	total := 1 + g.rng.IntN(20) + g.Char.AttackBonus()
	if total >= g.Foe.AC {
		dmg, err := Roll(g.rng, wandDice[g.Char.Wand])
		if err != nil {
			return "", err
		}
		g.Foe.HP -= dmg
		fmt.Fprintf(&b, "Tu touches (%d contre CA %d). %d dégâts.\n", total, g.Foe.AC, dmg)
		if g.Foe.HP <= 0 {
			b.WriteString(g.defeat("tué"))
			return b.String(), nil
		}
	} else {
		fmt.Fprintf(&b, "Tu rates (%d contre CA %d).\n", total, g.Foe.AC)
	}
	fmt.Fprintf(&b, "%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
	b.WriteString(g.monsterTurn(strike, breath, false))
	return b.String(), nil
}

// Flee rolls d20 plus the flee bonus against 12.
// Failure is one automatic melee hit, not the breath.
func (g *Game) Flee() (string, error) {
	if g.Foe == nil {
		return "", errors.New("Rien à fuir")
	}
	strike, _, err := g.prepare()
	if err != nil {
		return "", err
	}
	total := 1 + g.rng.IntN(20) + g.Char.FleeBonus()
	if total >= fleeDC {
		name := g.Foe.label()
		g.Foe = nil
		g.Shield = false
		return fmt.Sprintf("Tu sors (%d). %s reste.\n", total, name), nil
	}
	text, _ := g.melee(strike, true)
	return fmt.Sprintf("La fuite rate (%d).\n%s", total, text), nil
}

// Patronum works only on an Undead type. A wrong target does not spend the use.
func (g *Game) Patronum() (string, error) {
	if g.Foe == nil {
		return "", errors.New("Rien en face")
	}
	if !strings.EqualFold(g.Foe.Type, "undead") {
		return "", fmt.Errorf("%s n'est pas un mort-vivant", g.Foe.label())
	}
	if g.Char.PatronumSpent {
		return "", errors.New("Patronum déjà lancé depuis le dernier repos")
	}
	strike, _, err := g.prepare()
	if err != nil {
		return "", err
	}
	g.Char.PatronumSpent = true
	total := 1 + g.rng.IntN(20) + g.Char.AttackBonus()
	var b strings.Builder
	fmt.Fprintf(&b, "Patronum : %d contre CA %d.\n", total, g.Foe.AC)
	if total >= g.Foe.AC {
		b.WriteString(g.defeat("repoussé"))
		return b.String(), nil
	}
	text, _ := g.melee(strike, true)
	b.WriteString(text)
	return b.String(), nil
}

// Cast fetches the SRD spell, then applies the local rule for that slug.
// The DC of Incendio is fixed: the sheet has no caster ability.
func (g *Game) Cast(ctx context.Context, slug string) (string, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if !g.Char.Knows(slug) {
		return "", errors.New("Tu ne connais pas ce sort")
	}
	if g.Char.SpentSpell(slug) {
		return "", errors.New("Déjà lancé depuis le dernier repos")
	}
	if _, err := g.API.Spell(ctx, slug); err != nil {
		return "", err
	}
	if slug == "cure-wounds" {
		return g.castCure()
	}
	if g.Foe == nil {
		return "", errors.New("Pas de monstre en face")
	}
	switch slug {
	case "burning-hands", "shield", "magic-missile", "sleep":
	default:
		return "", errors.New("Sort inconnu")
	}
	strike, breath, err := g.prepare()
	if err != nil {
		return "", err
	}
	g.Char.spend(slug)
	switch slug {
	case "shield":
		g.Shield = true
		text := fmt.Sprintf("Protego. CA %d contre la prochaine attaque de mêlée.\n", playerAC+5)
		text += fmt.Sprintf("%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
		text += g.monsterTurn(strike, breath, false)
		return text, nil
	case "sleep":
		return g.castSleep(strike, breath)
	default:
		return g.castDamage(slug, strike, breath)
	}
}

// Heal restores 1d8+2 hit points. Everyone has it, once per rest.
// A foe still gets its riposte.
func (g *Game) Heal() (string, error) {
	if g.Char.SoinSpent {
		return "", errors.New("Tu t'es déjà soigné depuis le dernier repos")
	}
	if g.Char.HP >= g.Char.MaxHP {
		return "", errors.New("Tu es déjà au maximum")
	}
	var strike Strike
	var breath *Breath
	if g.Foe != nil {
		var err error
		strike, breath, err = g.prepare()
		if err != nil {
			return "", err
		}
	}
	n, err := Roll(g.rng, "1d8")
	if err != nil {
		return "", err
	}
	n += 2
	g.Char.SoinSpent = true
	before := g.Char.HP
	g.Char.HP += n
	if g.Char.HP > g.Char.MaxHP {
		g.Char.HP = g.Char.MaxHP
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Soin. +%d PV, tu as %d.\n", g.Char.HP-before, g.Char.HP)
	if g.Foe == nil {
		return b.String(), nil
	}
	fmt.Fprintf(&b, "%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
	b.WriteString(g.monsterTurn(strike, breath, false))
	return b.String(), nil
}

func (g *Game) castCure() (string, error) {
	var strike Strike
	var breath *Breath
	if g.Foe != nil {
		var err error
		strike, breath, err = g.prepare()
		if err != nil {
			return "", err
		}
	}
	n, err := Roll(g.rng, "1d8")
	if err != nil {
		return "", err
	}
	if g.Char.Wand == "licorne" {
		n += 2
	}
	g.Char.spend("cure-wounds")
	before := g.Char.HP
	g.Char.HP += n
	if g.Char.HP > g.Char.MaxHP {
		g.Char.HP = g.Char.MaxHP
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Episkey. +%d PV, tu as %d.\n", g.Char.HP-before, g.Char.HP)
	if g.Foe == nil {
		return b.String(), nil
	}
	fmt.Fprintf(&b, "%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
	b.WriteString(g.monsterTurn(strike, breath, false))
	return b.String(), nil
}

func (g *Game) castSleep(strike Strike, breath *Breath) (string, error) {
	var b strings.Builder
	if g.sleepImmune() {
		b.WriteString("Sommeil ne prend pas.\n")
		fmt.Fprintf(&b, "%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
		b.WriteString(g.monsterTurn(strike, breath, false))
		return b.String(), nil
	}
	roll, err := Roll(g.rng, "5d8")
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "Sommeil : %d. %s a %d PV.\n", roll, g.Foe.label(), g.Foe.HP)
	if g.Foe.HP <= roll {
		b.WriteString(g.defeat("endormi"))
		return b.String(), nil
	}
	b.WriteString("Il reste éveillé.\n")
	b.WriteString(g.monsterTurn(strike, breath, false))
	return b.String(), nil
}

func (g *Game) sleepImmune() bool {
	if strings.EqualFold(g.Foe.Type, "undead") {
		return true
	}
	return strings.Contains(strings.ToLower(g.Foe.Immunities), "charmed")
}

func (g *Game) castDamage(slug string, strike Strike, breath *Breath) (string, error) {
	var b strings.Builder
	switch slug {
	case "burning-hands":
		dmg, err := Roll(g.rng, "3d6")
		if err != nil {
			return "", err
		}
		save := 1 + g.rng.IntN(20)
		if save >= 12 {
			dmg /= 2
			fmt.Fprintf(&b, "Incendio : %d contre DD 12. Moitié, %d dégâts de feu.\n", save, dmg)
		} else {
			fmt.Fprintf(&b, "Incendio : %d contre DD 12. %d dégâts de feu.\n", save, dmg)
		}
		g.Foe.HP -= dmg
	case "magic-missile":
		var parts [3]int
		dmg := 0
		for i := 0; i < 3; i++ {
			n, err := Roll(g.rng, "1d4+1")
			if err != nil {
				return "", err
			}
			parts[i] = n
			dmg += n
		}
		fmt.Fprintf(&b, "Projectiles magiques : %d, %d, %d. %d dégâts.\n", parts[0], parts[1], parts[2], dmg)
		g.Foe.HP -= dmg
	default:
		return "", errors.New("Sort inconnu")
	}
	if g.Foe.HP <= 0 {
		b.WriteString(g.defeat("tué"))
		return b.String(), nil
	}
	fmt.Fprintf(&b, "%s : %d PV.\n", g.Foe.label(), g.Foe.HP)
	b.WriteString(g.monsterTurn(strike, breath, false))
	return b.String(), nil
}

func (g *Game) monsterTurn(strike Strike, breath *Breath, auto bool) string {
	text, died := g.melee(strike, auto)
	if died || auto || breath == nil || g.Foe == nil {
		return text
	}
	var b strings.Builder
	b.WriteString(text)
	if !g.Foe.BreathSpent {
		g.Foe.BreathSpent = true
		b.WriteString(g.applyBreath(*breath))
		return b.String()
	}
	roll := 1 + g.rng.IntN(6)
	if roll < breath.Recharge {
		fmt.Fprintf(&b, "Souffle pas prêt (%d).\n", roll)
		return b.String()
	}
	fmt.Fprintf(&b, "Souffle rechargé (%d).\n", roll)
	b.WriteString(g.applyBreath(*breath))
	return b.String()
}

func (g *Game) melee(strike Strike, auto bool) (string, bool) {
	if auto {
		dmg, err := rollSum(g.rng, strike.Parts)
		if err != nil {
			return fmt.Sprintf("%s : dés illisibles.\n", nomAttaque(strike.Name)), false
		}
		text, died := g.hurt(dmg)
		return fmt.Sprintf("%s : %d dégâts.\n%s", nomAttaque(strike.Name), dmg, text), died
	}
	ac := playerAC
	if g.Shield {
		ac += 5
	}
	g.Shield = false
	total := 1 + g.rng.IntN(20) + strike.Bonus
	if total < ac {
		return fmt.Sprintf("%s rate (%d contre CA %d).\n", nomAttaque(strike.Name), total, ac), false
	}
	dmg, err := rollSum(g.rng, strike.Parts)
	if err != nil {
		return fmt.Sprintf("%s : dés illisibles.\n", nomAttaque(strike.Name)), false
	}
	text, died := g.hurt(dmg)
	return fmt.Sprintf("%s : %d contre CA %d. %d dégâts.\n%s", nomAttaque(strike.Name), total, ac, dmg, text), died
}

func (g *Game) applyBreath(br Breath) string {
	roll := 1 + g.rng.IntN(20)
	dmg, err := Roll(g.rng, br.Dice)
	if err != nil {
		return "Souffle illisible.\n"
	}
	half := roll >= br.DC
	if half {
		dmg /= 2
	}
	text, _ := g.hurt(dmg)
	if half {
		return fmt.Sprintf("Souffle : %d contre DD %d. Moitié, %d dégâts de froid.\n%s", roll, br.DC, dmg, text)
	}
	return fmt.Sprintf("Souffle : %d contre DD %d. %d dégâts de froid.\n%s", roll, br.DC, dmg, text)
}

func (g *Game) hurt(n int) (string, bool) {
	if n < 0 {
		n = 0
	}
	g.Char.HP -= n
	if g.Char.HP > 0 {
		return fmt.Sprintf("Tu as %d PV.\n", g.Char.HP), false
	}
	g.Foe = nil
	g.Shield = false
	g.Char.Gaze = false
	g.Char.HP = 1
	g.Loc = g.World.Start
	g.Rested = false
	return "Tu te réveilles dans le Hall, à 1 PV.\n", true
}

func (g *Game) defeat(outcome string) string {
	foe := g.Foe
	lines := g.Char.gainXP(xpFor(foe.CR))
	var b strings.Builder
	switch outcome {
	case "tué":
		gold := 1 + g.rng.IntN(4)
		g.Char.Gold += gold
		g.Char.Kills[foe.Slug]++
		g.Char.note(foe.Slug, foe.label(), "tué")
		fmt.Fprintf(&b, "%s tombe. +%d PX, %d galions.\n", foe.label(), xpFor(foe.CR), gold)
		if foe.Slug == dragonSlug && !g.DragonDown {
			g.DragonDown = true
			b.WriteString("Le givre s'arrête. Le tronc est un tuyau cassé.\n")
		}
	case "endormi":
		g.Char.note(foe.Slug, foe.label(), "endormi")
		fmt.Fprintf(&b, "%s s'endort. +%d PX.\n", foe.label(), xpFor(foe.CR))
	case "repoussé":
		g.Char.note(foe.Slug, foe.label(), "repoussé")
		fmt.Fprintf(&b, "%s s'éloigne. +%d PX.\n", foe.label(), xpFor(foe.CR))
	}
	for _, ln := range lines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	g.Foe = nil
	g.Shield = false
	return b.String()
}
