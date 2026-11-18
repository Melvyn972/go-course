package repl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/VoluteTech/grimhold/internal/game"
	"github.com/VoluteTech/grimhold/internal/world"
)

type item struct {
	id    string
	label string
	blurb string
	art   string
	color string
}

func clearScreen() {
	if useColor {
		fmt.Print("\033[H\033[2J")
	}
}

func (r *REPL) runArrows(keys <-chan key, ticks <-chan struct{}) error {
	if useColor {
		fmt.Print("\033[?25l")
		defer fmt.Print("\033[?25h")
	}
	for {
		items := make([]item, len(r.order))
		for i, name := range r.order {
			items[i] = item{id: name, label: name, blurb: r.cmds[name].description}
		}
		idx, ok := choose(r.ctx, keys, ticks, r.prelude, r.banner, "Choisis", items, houseColor(r.g.Char.House), r.ambush)
		if !ok {
			fmt.Println()
			return nil
		}
		if err := r.activate(items[idx].id, keys, ticks); err != nil {
			if errors.Is(err, errQuit) {
				fmt.Println()
				return nil
			}
		}
	}
}

func (r *REPL) activate(name string, keys <-chan key, ticks <-chan struct{}) error {
	r.notice = ""
	switch name {
	case "combattre":
		return r.menuFight(keys, ticks)
	case "aller":
		return r.menuTravel(keys, ticks)
	case "lancer":
		return r.menuCast(keys, ticks)
	case "examiner":
		return r.menuInspect(keys, ticks)
	case "bestiaire":
		return r.menuBestiary(keys, ticks)
	case "quitter":
		return errQuit
	default:
		clearScreen()
		err := r.cmds[name].callback(nil)
		if err != nil && !errors.Is(err, errQuit) {
			fmt.Println(err.Error())
		}
		if errors.Is(err, errQuit) {
			return err
		}
		r.waitEnter(keys, ticks)
		return nil
	}
}

func (r *REPL) menuFight(keys <-chan key, ticks <-chan struct{}) error {
	if r.g.Foe == nil {
		clearScreen()
		fmt.Println("Rien en face")
		r.waitEnter(keys, ticks)
		return nil
	}
	for r.g.Foe != nil {
		items := []item{{id: "frapper", label: "Frapper", blurb: "Un coup avec la baguette."}}
		for _, s := range r.g.Char.Spells {
			state := "prêt"
			if r.g.Char.SpentSpell(s) {
				state = "dépensé"
			}
			items = append(items, item{
				id:    "sort:" + s,
				label: game.SpellTitle(s) + " — " + state,
				blurb: "Lance ce sort sur le monstre.",
			})
		}
		pstate := "prêt"
		if r.g.Char.PatronumSpent {
			pstate = "dépensé"
		}
		sstate := "prêt"
		if r.g.Char.SoinSpent {
			sstate = "dépensé"
		}
		items = append(items,
			item{id: "patronum", label: "Patronum — " + pstate, blurb: "Repousse un mort-vivant."},
			item{id: "soigner", label: "Soigner — " + sstate, blurb: "Rend 1d8+2 PV. Le monstre riposte."},
			item{id: "fuir", label: "Fuir", blurb: "Quitte le combat."},
			item{label: "Retour"},
		)
		idx, ok := choose(r.ctx, keys, ticks, r.prelude, r.banner, "Combat", items, houseColor(r.g.Char.House), r.ambush)
		if !ok || items[idx].id == "" {
			return nil
		}
		clearScreen()
		if err := r.fightAction(items[idx].id); err != nil {
			fmt.Println(err.Error())
		}
		r.waitEnter(keys, ticks)
	}
	return nil
}

func (r *REPL) fightAction(id string) error {
	slug := ""
	if r.g.Foe != nil {
		slug = r.g.Foe.Slug
	}
	switch {
	case id == "frapper":
		msg, err := r.g.Fight()
		if err != nil {
			return err
		}
		r.showSlug(slug, msg)
	case id == "fuir":
		msg, err := r.g.Flee()
		if err != nil {
			return err
		}
		r.showSlug(slug, msg)
	case id == "soigner":
		msg, err := r.g.Heal()
		if err != nil {
			return err
		}
		fmt.Println(paint(spellColor("cure-wounds"), spellArt("cure-wounds")))
		r.showSlug(slug, msg)
	case id == "patronum":
		msg, err := r.g.Patronum()
		if err != nil {
			return err
		}
		r.showSlug(slug, msg)
	case strings.HasPrefix(id, "sort:"):
		spell := strings.TrimPrefix(id, "sort:")
		msg, err := r.g.Cast(r.ctx, spell)
		if err != nil {
			return err
		}
		fmt.Println(paint(spellColor(spell), spellArt(spell)))
		r.showSlug(slug, msg)
	default:
		return errors.New("Action inconnue")
	}
	return nil
}

func (r *REPL) menuTravel(keys <-chan key, ticks <-chan struct{}) error {
	exits, err := r.g.Exits()
	if err != nil {
		clearScreen()
		fmt.Println(err.Error())
		r.waitEnter(keys, ticks)
		return nil
	}
	items := make([]item, 0, len(exits)+1)
	for _, e := range exits {
		it := item{id: e.ID, label: e.Name, art: sceneArt(e.ID), color: sceneColor(e.ID)}
		if dest, ok := r.g.World.Get(e.ID); ok {
			it.blurb = dest.Description
		}
		items = append(items, it)
	}
	items = append(items, item{label: "Retour"})
	idx, ok := choose(r.ctx, keys, ticks, nil, r.banner, "Où aller", items, houseColor(r.g.Char.House), r.ambush)
	if !ok || items[idx].id == "" {
		return nil
	}
	clearScreen()
	msg, err := r.g.Travel(items[idx].id)
	if err != nil {
		fmt.Println(err.Error())
		r.waitEnter(keys, ticks)
		return nil
	}
	if r.reset != nil {
		r.reset()
	}
	r.showHere(msg)
	r.waitEnter(keys, ticks)
	return nil
}

func (r *REPL) menuCast(keys <-chan key, ticks <-chan struct{}) error {
	known := r.g.Char.Spells
	items := make([]item, 0, len(known)+1)
	for _, s := range known {
		state := "prêt"
		if r.g.Char.SpentSpell(s) {
			state = "dépensé"
		}
		items = append(items, item{
			id:    s,
			label: game.SpellTitle(s) + " — " + state,
			art:   spellArt(s),
			color: spellColor(s),
		})
	}
	items = append(items, item{label: "Retour"})
	idx, ok := choose(r.ctx, keys, ticks, nil, r.banner, "Quel sort", items, houseColor(r.g.Char.House), r.ambush)
	if !ok || items[idx].id == "" {
		return nil
	}
	clearScreen()
	slug := items[idx].id
	fmt.Println(paint(spellColor(slug), spellArt(slug)))
	foe := ""
	if r.g.Foe != nil {
		foe = r.g.Foe.Slug
	}
	msg, err := r.g.Cast(r.ctx, slug)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		r.showSlug(foe, msg)
	}
	r.waitEnter(keys, ticks)
	return nil
}

func (r *REPL) menuInspect(keys <-chan key, ticks <-chan struct{}) error {
	reg, ok := r.g.World.Get(r.g.Loc)
	items := []item{}
	if ok {
		seen := map[string]bool{}
		if r.g.Foe != nil && !seen[r.g.Foe.Slug] {
			seen[r.g.Foe.Slug] = true
			items = append(items, foeItem(r.g.Foe.Slug))
		}
		for _, slug := range reg.Foes {
			if seen[slug] {
				continue
			}
			seen[slug] = true
			items = append(items, foeItem(slug))
		}
	}
	items = append(items, item{label: "Retour"})
	idx, ok := choose(r.ctx, keys, ticks, nil, r.banner, "Examiner", items, houseColor(r.g.Char.House), r.ambush)
	if !ok || items[idx].id == "" {
		return nil
	}
	clearScreen()
	fmt.Println(paint(items[idx].color, items[idx].art))
	msg, err := r.g.Inspect(r.ctx, items[idx].id)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		r.say(msg)
	}
	r.waitEnter(keys, ticks)
	return nil
}

func foeItem(slug string) item {
	return item{
		id:    slug,
		label: game.Label(slug, slug),
		art:   foeArt(slug),
		color: foeColor(slug),
	}
}

func (r *REPL) menuBestiary(keys <-chan key, ticks <-chan struct{}) error {
	items := []item{
		{id: "", label: "Voir la page"},
		{id: "next", label: "Page suivante"},
		{id: "prev", label: "Page précédente"},
		{label: "Retour"},
	}
	idx, ok := choose(r.ctx, keys, ticks, r.prelude, r.banner, "Bestiaire", items, houseColor(r.g.Char.House), r.ambush)
	if !ok || idx == len(items)-1 {
		return nil
	}
	clearScreen()
	msg, err := r.g.Bestiary(r.ctx, items[idx].id)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		r.say(msg)
	}
	r.waitEnter(keys, ticks)
	return nil
}

func (r *REPL) prelude() string {
	var b strings.Builder
	c := r.g.Char
	fmt.Fprintf(&b, "%s\n", hpBar(c.HP, c.MaxHP, houseColor(c.House)))
	if pic := sceneArt(r.g.Loc); pic != "" {
		fmt.Fprintf(&b, "%s\n", paint(sceneColor(r.g.Loc), pic))
	}
	if r.g.Foe != nil {
		if pic := foeArt(r.g.Foe.Slug); pic != "" {
			fmt.Fprintf(&b, "%s\n", paint(foeColor(r.g.Foe.Slug), pic))
		}
	}
	if msg, err := r.g.Look(); err == nil {
		b.WriteString(colorizeDanger(msg))
	}
	return b.String()
}

func (r *REPL) ambush() {
	if !r.g.CanAmbush() {
		return
	}
	msg, err := r.g.Ambush(r.ctx)
	if err != nil || msg == "" {
		return
	}
	r.notice = strings.TrimSpace(msg)
}

func (r *REPL) banner() string { return r.notice }

func (r *REPL) showScene() {
	if pic := sceneArt(r.g.Loc); pic != "" {
		fmt.Println(paint(sceneColor(r.g.Loc), pic))
	}
}

func (r *REPL) showSlug(slug, msg string) {
	r.showScene()
	if pic := foeArt(slug); pic != "" {
		fmt.Println(paint(foeColor(slug), pic))
	}
	r.say(msg)
}

func (r *REPL) waitEnter(keys <-chan key, ticks <-chan struct{}) {
	fmt.Println()
	fmt.Println("Entrée pour continuer")
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticks:
			r.ambush()
		case k, ok := <-keys:
			if !ok || k.kind == 'E' {
				return
			}
		}
	}
}

func choose(ctx context.Context, keys <-chan key, ticks <-chan struct{}, prelude func() string, banner func() string, title string, items []item, hi string, onTick func()) (int, bool) {
	if len(items) == 0 {
		return 0, false
	}
	sel := 0
	draw := func() {
		clearScreen()
		if items[sel].art != "" {
			fmt.Println(paint(items[sel].color, items[sel].art))
		} else if prelude != nil {
			fmt.Print(prelude())
		}
		if banner != nil && banner() != "" {
			fmt.Println(banner())
			fmt.Println()
		}
		fmt.Println(title)
		fmt.Println("Flèches pour choisir. Entrée pour valider.")
		fmt.Println()
		for i, it := range items {
			if i == sel {
				col := hi
				if it.color != "" {
					col = it.color
				}
				fmt.Printf("  > %s\n", paint(col, it.label))
				if it.blurb != "" {
					fmt.Printf("    %s\n", it.blurb)
				}
				continue
			}
			fmt.Printf("    %s\n", it.label)
		}
	}
	draw()
	for {
		select {
		case <-ctx.Done():
			return 0, false
		case <-ticks:
			if onTick != nil {
				onTick()
			}
			draw()
		case k, ok := <-keys:
			if !ok {
				return 0, false
			}
			switch k.kind {
			case 'U':
				sel--
				if sel < 0 {
					sel = len(items) - 1
				}
			case 'D':
				sel++
				if sel >= len(items) {
					sel = 0
				}
			case 'E':
				return sel, true
			default:
				continue
			}
			draw()
		}
	}
}

func (r *REPL) paintMap() {
	for i := range r.g.World.Regions {
		reg := &r.g.World.Regions[i]
		r.paintRegion(reg)
	}
}

func (r *REPL) paintRegion(reg *world.Region) {
	if pic := sceneArt(reg.ID); pic != "" {
		fmt.Println(paint(sceneColor(reg.ID), pic))
	}
	here := ""
	if reg.ID == r.g.Loc {
		here = " ← ici"
	}
	fmt.Printf("%s — danger %d, %s%s\n", reg.Name, reg.Danger, game.DangerText(reg.Danger), here)
	fmt.Printf("  %s\n", reg.Description)
	parts := make([]string, len(reg.Exits))
	for i, id := range reg.Exits {
		if dest, ok := r.g.World.Get(id); ok {
			parts[i] = dest.Name
		} else {
			parts[i] = id
		}
	}
	if len(parts) == 0 {
		fmt.Println("  sorties : aucune")
	} else {
		fmt.Printf("  sorties : %s\n", strings.Join(parts, ", "))
	}
	if reg.ID == "iron-vault" {
		fmt.Println("  entrée : niveau 4")
	}
	fmt.Println()
}
