// Package repl is the command loop. Help is built from the command map.
package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/VoluteTech/grimhold/internal/game"
)

const ambushEvery = 45 * time.Second

type cliCommand struct {
	name        string
	description string
	callback    func(args []string) error
}

// REPL holds the command map and the session it dispatches into.
type REPL struct {
	ctx    context.Context
	g      *game.Game
	cmds   map[string]cliCommand
	order  []string
	reset  func()
	notice string
}

func newREPL(ctx context.Context, g *game.Game, reset func()) *REPL {
	r := &REPL{
		ctx:   ctx,
		g:     g,
		cmds:  map[string]cliCommand{},
		reset: reset,
	}
	r.add("aide", "Affiche ce message", r.cmdHelp)
	r.add("carte", "Liste les régions, leur danger et leurs sorties", r.cmdMap)
	r.add("aller", "Entre dans une sortie. Sans argument, liste les numéros", r.cmdTravel)
	r.add("regarder", "Décrit la pièce, les sorties et le danger", r.cmdLook)
	r.add("explorer", "Cherche un monstre ici", r.cmdExplore)
	r.add("combattre", "Ouvre le combat : coup, sorts, soin ou fuite", r.cmdFight)
	r.add("fuir", "Tente de partir", r.cmdFlee)
	r.add("reposer", "Soin complet, une fois par entrée", r.cmdRest)
	r.add("soigner", "Rend 1d8+2 PV, une fois par repos", r.cmdHeal)
	r.add("fiche", "Fiche du personnage", r.cmdCharacter)
	r.add("lancer", "Lance un sort connu. Sans argument, liste les numéros", r.cmdCast)
	r.add("patronum", "Repousse un mort-vivant, une fois par repos", r.cmdPatronum)
	r.add("examiner", "Fiche d'un monstre", r.cmdInspect)
	r.add("bestiaire", "Page de monstres. Puis suivant ou précédent", r.cmdBestiary)
	r.add("journal", "Tués, endormis, repoussés", r.cmdJournal)
	r.add("sauver", "Écrit la sauvegarde", r.cmdSave)
	r.add("charger", "Relit la sauvegarde", r.cmdLoad)
	r.add("quitter", "Quitte", r.cmdQuit)
	r.alias("aide", "help")
	r.alias("carte", "map")
	r.alias("aller", "travel")
	r.alias("regarder", "look")
	r.alias("explorer", "explore")
	r.alias("combattre", "fight")
	r.alias("fuir", "flee")
	r.alias("reposer", "rest")
	r.alias("fiche", "character")
	r.alias("lancer", "cast")
	r.alias("examiner", "inspect")
	r.alias("bestiaire", "bestiary")
	r.alias("sauver", "save")
	r.alias("charger", "load")
	r.alias("quitter", "quit")
	return r
}

func (r *REPL) alias(visible, hidden string) {
	cmd := r.cmds[visible]
	cmd.name = visible
	r.cmds[hidden] = cmd
}

func (r *REPL) add(name, desc string, fn func([]string) error) {
	r.cmds[name] = cliCommand{name: name, description: desc, callback: fn}
	r.order = append(r.order, name)
}

// Run reads lines until quit, EOF, or ctx cancellation.
// On a console it draws a menu: arrows move, Enter validates.
// A goroutine ticks every 45s. select takes either input or an ambush.
func Run(ctx context.Context, sc *bufio.Scanner, g *game.Game) error {
	if isTerm(os.Stdout) {
		enableVT()
		useColor = true
	}
	ticks := make(chan struct{}, 1)
	resets := make(chan struct{}, 1)
	go ambushLoop(ctx, ticks, resets)
	r := newREPL(ctx, g, func() {
		select {
		case resets <- struct{}{}:
		default:
		}
	})
	if isTerm(os.Stdin) && isTerm(os.Stdout) {
		if keys, ok := consoleKeys(); ok {
			return r.runArrows(keys, ticks)
		}
	}
	return r.runLines(sc, ticks)
}

func (r *REPL) runLines(sc *bufio.Scanner, ticks <-chan struct{}) error {
	lines := make(chan string)
	go func() {
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	fmt.Println("Un numéro, ou une commande. aide pour la liste.")
	if msg, err := r.g.Look(); err == nil {
		r.showHere(msg)
	}
	for {
		r.prompt()
		select {
		case <-r.ctx.Done():
			fmt.Println()
			return nil
		case line, ok := <-lines:
			if !ok {
				fmt.Println()
				return nil
			}
			if !r.dispatch(line) {
				fmt.Println()
				return nil
			}
		case <-ticks:
			if !r.g.CanAmbush() {
				continue
			}
			fmt.Println()
			msg, err := r.g.Ambush(r.ctx)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			if msg == "" {
				continue
			}
			r.showFoe(msg)
			// Drop a line only if Enter was already pressed. A command typed
			// after the spark is kept.
			select {
			case line, ok := <-lines:
				if ok && strings.TrimSpace(line) != "" {
					fmt.Println("La ligne en cours est perdue.")
				}
			default:
			}
		}
	}
}

func ambushLoop(ctx context.Context, ticks, resets chan struct{}) {
	timer := time.NewTimer(ambushEvery)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-resets:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(ambushEvery)
		case <-timer.C:
			select {
			case ticks <- struct{}{}:
			default:
			}
			timer.Reset(ambushEvery)
		}
	}
}

func (r *REPL) prompt() {
	c := r.g.Char
	fmt.Printf("%s grimhold > ", hpBar(c.HP, c.MaxHP, houseColor(c.House)))
}

func (r *REPL) say(s string) { fmt.Print(colorizeDanger(s)) }

func (r *REPL) showHere(msg string) {
	if pic := sceneArt(r.g.Loc); pic != "" {
		fmt.Println(paint(sceneColor(r.g.Loc), pic))
	}
	if r.g.Foe != nil {
		if pic := foeArt(r.g.Foe.Slug); pic != "" {
			fmt.Println(paint(foeColor(r.g.Foe.Slug), pic))
		}
	}
	r.say(msg)
}

func (r *REPL) showFoe(msg string) {
	r.showScene()
	if r.g.Foe != nil {
		if pic := foeArt(r.g.Foe.Slug); pic != "" {
			fmt.Println(paint(foeColor(r.g.Foe.Slug), pic))
		}
	}
	r.say(msg)
}

func (r *REPL) dispatch(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return true
	}
	name := strings.ToLower(fields[0])
	if n, err := strconv.Atoi(name); err == nil {
		if n < 1 || n > len(r.order) {
			fmt.Println("Numéro hors liste. aide pour la liste.")
			return true
		}
		name = r.order[n-1]
		fields[0] = name
	}
	cmd, ok := r.cmds[name]
	if !ok {
		msg, err := r.g.Travel(strings.Join(fields, " "))
		if err == nil {
			if r.reset != nil {
				r.reset()
			}
			r.showHere(msg)
			return true
		}
		if !strings.HasPrefix(err.Error(), "Région") {
			fmt.Println(err.Error())
			return true
		}
		fmt.Println("Commande inconnue. aide pour la liste.")
		return true
	}
	err := cmd.callback(fields[1:])
	if errors.Is(err, errQuit) {
		return false
	}
	if err != nil {
		fmt.Println(err.Error())
	}
	return true
}

var errQuit = errors.New("quit")

func (r *REPL) cmdHelp(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : aide")
	}
	fmt.Println("Grimhold. Le tronc de cheminette est fendu.")
	fmt.Println()
	fmt.Println("Flèches et Entrée dans le menu. Le numéro devant le nom marche aussi au clavier.")
	width := 0
	for _, name := range r.order {
		if len(name) > width {
			width = len(name)
		}
	}
	for i, name := range r.order {
		cmd := r.cmds[name]
		fmt.Printf(" %2d  %-*s  %s\n", i+1, width, cmd.name, cmd.description)
	}
	return nil
}

func (r *REPL) cmdMap(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : carte")
	}
	r.paintMap()
	return nil
}

func (r *REPL) cmdTravel(args []string) error {
	if len(args) < 1 {
		exits, err := r.g.Exits()
		if err != nil {
			return err
		}
		fmt.Println("Sorties :")
		for i, e := range exits {
			fmt.Printf("  %d  %s\n", i+1, e.Name)
		}
		return nil
	}
	q := strings.Join(args, " ")
	if len(args) == 1 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			exits, err := r.g.Exits()
			if err != nil {
				return err
			}
			if n < 1 || n > len(exits) {
				return errors.New("Numéro hors liste")
			}
			q = exits[n-1].ID
		}
	}
	msg, err := r.g.Travel(q)
	if err != nil {
		return err
	}
	if r.reset != nil {
		r.reset()
	}
	r.showHere(msg)
	return nil
}

func (r *REPL) cmdLook(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : regarder")
	}
	msg, err := r.g.Look()
	if err != nil {
		return err
	}
	r.showHere(msg)
	return nil
}

func (r *REPL) cmdExplore(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : explorer")
	}
	msg, err := r.g.Explore(r.ctx)
	if err != nil {
		return err
	}
	r.showFoe(msg)
	return nil
}

func (r *REPL) cmdFight(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : combattre")
	}
	slug := ""
	if r.g.Foe != nil {
		slug = r.g.Foe.Slug
	}
	msg, err := r.g.Fight()
	if err != nil {
		return err
	}
	r.showSlug(slug, msg)
	return nil
}

func (r *REPL) cmdFlee(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : fuir")
	}
	slug := ""
	if r.g.Foe != nil {
		slug = r.g.Foe.Slug
	}
	msg, err := r.g.Flee()
	if err != nil {
		return err
	}
	r.showSlug(slug, msg)
	return nil
}

func (r *REPL) cmdHeal(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : soigner")
	}
	slug := ""
	if r.g.Foe != nil {
		slug = r.g.Foe.Slug
	}
	msg, err := r.g.Heal()
	if err != nil {
		return err
	}
	if pic := spellArt("cure-wounds"); pic != "" {
		fmt.Println(paint(spellColor("cure-wounds"), pic))
	}
	r.showSlug(slug, msg)
	return nil
}

func (r *REPL) cmdRest(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : reposer")
	}
	msg, err := r.g.Rest()
	if err != nil {
		return err
	}
	r.showScene()
	r.say(msg)
	return nil
}

func (r *REPL) cmdCharacter(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : fiche")
	}
	c := r.g.Char
	if pic := houseArt(c.House); pic != "" {
		fmt.Println(paint(houseColor(c.House), pic))
	}
	if pic := wandArt(c.Wand); pic != "" {
		fmt.Println(paint(wandColor(c.Wand), pic))
	}
	for _, s := range c.Spells {
		if pic := spellArt(s); pic != "" {
			fmt.Println(paint(spellColor(s), pic))
		}
	}
	fmt.Println(hpBar(c.HP, c.MaxHP, houseColor(c.House)))
	r.say(r.g.Sheet())
	return nil
}

func (r *REPL) cmdCast(args []string) error {
	known := r.g.Char.Spells
	if len(args) == 0 {
		fmt.Println("Sorts :")
		for i, s := range known {
			state := "prêt"
			if r.g.Char.SpentSpell(s) {
				state = "dépensé"
			}
			fmt.Printf("  %d  %s — %s\n%s\n", i+1, game.SpellTitle(s), state, spellArt(s))
		}
		return nil
	}
	slug := strings.ToLower(strings.Join(args, "-"))
	if len(args) == 1 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			if n < 1 || n > len(known) {
				return errors.New("Numéro hors liste")
			}
			slug = known[n-1]
		}
	}
	foe := ""
	if r.g.Foe != nil {
		foe = r.g.Foe.Slug
	}
	msg, err := r.g.Cast(r.ctx, slug)
	if err != nil {
		return err
	}
	fmt.Println(paint(spellColor(slug), spellArt(slug)))
	r.showSlug(foe, msg)
	return nil
}

func (r *REPL) cmdPatronum(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : patronum")
	}
	slug := ""
	if r.g.Foe != nil {
		slug = r.g.Foe.Slug
	}
	msg, err := r.g.Patronum()
	if err != nil {
		return err
	}
	r.showSlug(slug, msg)
	return nil
}

func (r *REPL) cmdInspect(args []string) error {
	if len(args) < 1 {
		return errors.New("usage : examiner <monstre>")
	}
	q := strings.Join(args, " ")
	msg, err := r.g.Inspect(r.ctx, q)
	if err != nil {
		return err
	}
	slug := game.CanonSlug(q)
	if pic := foeArt(slug); pic != "" {
		fmt.Println(paint(foeColor(slug), pic))
	}
	r.say(msg)
	return nil
}

func (r *REPL) cmdBestiary(args []string) error {
	which := ""
	switch len(args) {
	case 0:
	case 1:
		switch strings.ToLower(args[0]) {
		case "suivant", "next":
			which = "next"
		case "précédent", "precedent", "prev":
			which = "prev"
		default:
			return errors.New("usage : bestiaire [suivant|précédent]")
		}
	default:
		return errors.New("usage : bestiaire [suivant|précédent]")
	}
	msg, err := r.g.Bestiary(r.ctx, which)
	if err != nil {
		return err
	}
	r.say(msg)
	return nil
}

func (r *REPL) cmdJournal(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : journal")
	}
	if len(r.g.Char.Journal) == 0 {
		r.say(r.g.Journal())
		return nil
	}
	for _, e := range r.g.Char.Journal {
		if pic := foeArt(e.Slug); pic != "" {
			fmt.Println(paint(foeColor(e.Slug), pic))
		}
		fmt.Printf("%s — %s\n", e.Name, e.Outcome)
	}
	return nil
}

func (r *REPL) cmdSave(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : sauver")
	}
	if err := r.g.Save("save.json"); err != nil {
		return err
	}
	fmt.Println("Sauvegarde écrite.")
	return nil
}

func (r *REPL) cmdLoad(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : charger")
	}
	if err := r.g.Load(r.ctx, "save.json"); err != nil {
		return err
	}
	fmt.Println("Sauvegarde chargée.")
	msg, err := r.g.Look()
	if err != nil {
		return err
	}
	r.showHere(msg)
	return nil
}

func (r *REPL) cmdQuit(args []string) error {
	if len(args) != 0 {
		return errors.New("usage : quitter")
	}
	return errQuit
}
