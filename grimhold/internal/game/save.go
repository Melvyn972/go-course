package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type fileState struct {
	Location   string    `json:"location"`
	Rested     bool      `json:"rested"`
	DragonDown bool      `json:"dragonDown"`
	Shield     bool      `json:"shield"`
	Character  Character `json:"character"`
	Foe        *savedFoe `json:"foe,omitempty"`
	Next       string    `json:"bestiaryNext,omitempty"`
	Prev       string    `json:"bestiaryPrev,omitempty"`
}

type savedFoe struct {
	Slug        string `json:"slug"`
	HP          int    `json:"hp"`
	BreathSpent bool   `json:"breathSpent"`
}

// Save writes the session. Callers must not call it from death or from a
// failed explore: those paths only change memory.
func (g *Game) Save(path string) error {
	f := fileState{
		Location:   g.Loc,
		Rested:     g.Rested,
		DragonDown: g.DragonDown,
		Shield:     g.Shield,
		Character:  g.Char,
		Next:       g.Next,
		Prev:       g.Prev,
	}
	if g.Foe != nil {
		f.Foe = &savedFoe{Slug: g.Foe.Slug, HP: g.Foe.HP, BreathSpent: g.Foe.BreathSpent}
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage: %w", err)
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("écriture de %s: %w", path, err)
	}
	bak := path + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(path, bak); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmp)
		return fmt.Errorf("écriture de %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Rename(bak, path)
		return fmt.Errorf("écriture de %s: %w", path, err)
	}
	_ = os.Remove(bak)
	return nil
}

// Load replaces the session. A missing file is ErrNoSave.
// The foe is fetched again; current HP and breath state are kept.
func (g *Game) Load(ctx context.Context, path string) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoSave
	}
	if err != nil {
		return fmt.Errorf("lecture de %s: %w", path, err)
	}
	var f fileState
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("sauvegarde illisible: %w", err)
	}
	if err := f.Character.validate(); err != nil {
		return err
	}
	f.Character.sync()
	if _, ok := g.World.Get(f.Location); !ok {
		return fmt.Errorf("lieu sauvegardé inconnu: %s", f.Location)
	}
	var foe *Foe
	if f.Foe != nil {
		m, err := g.API.Monster(ctx, f.Foe.Slug)
		if err != nil {
			return fmt.Errorf("reprise du monstre: %w", err)
		}
		foe = foeFrom(m)
		foe.HP = f.Foe.HP
		foe.BreathSpent = f.Foe.BreathSpent
	}
	g.Loc = f.Location
	g.Rested = f.Rested
	g.DragonDown = f.DragonDown
	g.Shield = f.Shield
	g.Char = f.Character
	g.Foe = foe
	g.Next = f.Next
	g.Prev = f.Prev
	g.ready = true
	return nil
}
