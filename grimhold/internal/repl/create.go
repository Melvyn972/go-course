package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/VoluteTech/grimhold/internal/game"
)

type choice struct {
	id    string
	title string
	blurb string
	art   string
	also  []string
}

// Create asks for house, wand, and two spells. Each choice is a number,
// with a picture and what that choice changes.
func Create(ctx context.Context, sc *bufio.Scanner) (house, wand string, spells []string, err error) {
	if isTerm(os.Stdout) {
		enableVT()
		useColor = true
	}
	if isTerm(os.Stdin) && isTerm(os.Stdout) {
		if keys, ok := consoleKeys(); ok {
			return createArrows(ctx, keys)
		}
	}
	fmt.Println("Trois choix. Tape le numéro. Le dessin et la ligne disent ce que ça change.")
	house, err = pick(sc, "Maison", []choice{
		{id: "gryffondor", title: game.HouseTitle("gryffondor"), blurb: "+1 à l'attaque.", art: houseArt("gryffondor")},
		{id: "serdaigle", title: game.HouseTitle("serdaigle"), blurb: "+2 au jet de fuite.", art: houseArt("serdaigle")},
		{id: "poufsouffle", title: game.HouseTitle("poufsouffle"), blurb: "+2 aux PV maximum.", art: houseArt("poufsouffle")},
		{id: "serpentard", title: game.HouseTitle("serpentard"), blurb: "Quand un monstre apparaît, sa CA et son FP s'affichent.", art: houseArt("serpentard")},
	})
	if err != nil {
		return "", "", nil, err
	}
	wand, err = pick(sc, "Baguette", []choice{
		{id: "phenix", title: game.WandTitle("phenix"), blurb: "Frappe en 1d8.", art: wandArt("phenix")},
		{id: "dragon", title: game.WandTitle("dragon"), blurb: "Frappe en 1d6 et +1 à l'attaque.", art: wandArt("dragon")},
		{id: "licorne", title: game.WandTitle("licorne"), blurb: "Frappe en 1d4. Episkey soigne 2 PV de plus. Sans ce sort, ce +2 n'existe pas.", art: wandArt("licorne")},
	})
	if err != nil {
		return "", "", nil, err
	}
	first, err := pick(sc, "Premier sort", spellChoices(nil))
	if err != nil {
		return "", "", nil, err
	}
	second, err := pick(sc, "Deuxième sort", spellChoices(map[string]bool{first: true}))
	if err != nil {
		return "", "", nil, err
	}
	return house, wand, []string{first, second}, nil
}

func createArrows(ctx context.Context, keys <-chan key) (house, wand string, spells []string, err error) {
	if useColor {
		fmt.Print("\033[?25l")
		defer fmt.Print("\033[?25h")
	}
	houses := []choice{
		{id: "gryffondor", title: game.HouseTitle("gryffondor"), blurb: "+1 à l'attaque.", art: houseArt("gryffondor")},
		{id: "serdaigle", title: game.HouseTitle("serdaigle"), blurb: "+2 au jet de fuite.", art: houseArt("serdaigle")},
		{id: "poufsouffle", title: game.HouseTitle("poufsouffle"), blurb: "+2 aux PV maximum.", art: houseArt("poufsouffle")},
		{id: "serpentard", title: game.HouseTitle("serpentard"), blurb: "Quand un monstre apparaît, sa CA et son FP s'affichent.", art: houseArt("serpentard")},
	}
	wands := []choice{
		{id: "phenix", title: game.WandTitle("phenix"), blurb: "Frappe en 1d8.", art: wandArt("phenix")},
		{id: "dragon", title: game.WandTitle("dragon"), blurb: "Frappe en 1d6 et +1 à l'attaque.", art: wandArt("dragon")},
		{id: "licorne", title: game.WandTitle("licorne"), blurb: "Frappe en 1d4. Episkey soigne 2 PV de plus. Sans ce sort, ce +2 n'existe pas.", art: wandArt("licorne")},
	}
	house, err = arrowChoice(ctx, keys, "Maison", houses)
	if err != nil {
		return "", "", nil, err
	}
	wand, err = arrowChoice(ctx, keys, "Baguette", wands)
	if err != nil {
		return "", "", nil, err
	}
	first, err := arrowChoice(ctx, keys, "Premier sort", spellChoices(nil))
	if err != nil {
		return "", "", nil, err
	}
	second, err := arrowChoice(ctx, keys, "Deuxième sort", spellChoices(map[string]bool{first: true}))
	if err != nil {
		return "", "", nil, err
	}
	return house, wand, []string{first, second}, nil
}

func arrowChoice(ctx context.Context, keys <-chan key, heading string, opts []choice) (string, error) {
	items := make([]item, len(opts))
	for i, o := range opts {
		col := spellColor(o.id)
		switch heading {
		case "Maison":
			col = houseColor(o.id)
		case "Baguette":
			col = wandColor(o.id)
		}
		items[i] = item{id: o.id, label: o.title, blurb: o.blurb, art: o.art, color: col}
	}
	idx, ok := choose(ctx, keys, nil, nil, nil, heading, items, "", nil)
	if !ok {
		return "", io.EOF
	}
	return items[idx].id, nil
}

func spellChoices(skip map[string]bool) []choice {
	blurbs := map[string]string{
		"burning-hands": "3d6 de feu. Le monstre jette un d20 contre 12. Moitié si le jet passe. Il faut un monstre en face.",
		"shield":        "+5 de CA contre la prochaine attaque de corps à corps. Le souffle n'est pas couvert.",
		"cure-wounds":   "Rend 1d8 PV. La licorne ajoute +2. Pas besoin de monstre.",
		"sleep":         "Endort si les PV tiennent dans 5d8. Rien sur un mort-vivant, ni si charmé est immunisé. Le sort est quand même dépensé.",
		"magic-missile": "Trois traits de 1d4+1. Ils touchent. Il faut un monstre en face.",
	}
	also := map[string][]string{
		"burning-hands": {"incendio"},
		"shield":        {"protego"},
		"cure-wounds":   {"episkey"},
		"sleep":         {"sommeil"},
		"magic-missile": {"projectiles-magiques", "projectiles"},
	}
	var out []choice
	for _, id := range game.SpellOrder() {
		if skip[id] {
			continue
		}
		out = append(out, choice{
			id:    id,
			title: game.SpellTitle(id),
			blurb: blurbs[id],
			art:   spellArt(id),
			also:  also[id],
		})
	}
	return out
}

func pick(sc *bufio.Scanner, heading string, opts []choice) (string, error) {
	fmt.Println()
	fmt.Println(heading)
	for i, o := range opts {
		title := o.title
		art := o.art
		switch heading {
		case "Maison":
			title = paint(houseColor(o.id), o.title)
			art = paint(houseColor(o.id), o.art)
		case "Baguette":
			art = paint(wandColor(o.id), o.art)
		default:
			art = paint(spellColor(o.id), o.art)
		}
		fmt.Printf("\n%d  %s\n   %s\n%s\n", i+1, title, o.blurb, art)
	}
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return "", err
			}
			return "", io.EOF
		}
		v := strings.ToLower(strings.TrimSpace(sc.Text()))
		v = strings.Join(strings.Fields(v), "-")
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= len(opts) {
			return opts[n-1].id, nil
		}
		for _, o := range opts {
			if v == o.id || v == strings.ToLower(o.title) {
				return o.id, nil
			}
			for _, a := range o.also {
				if v == a {
					return o.id, nil
				}
			}
		}
		fmt.Println("Choix invalide.")
	}
}
