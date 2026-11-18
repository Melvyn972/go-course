// Package world loads the region graph from world.json.
package world

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// Region is one node of the map. Foes are Open5e slugs.
type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
}

// World is the authored map. Start is a region id.
type World struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
	byID    map[string]*Region
}

// Load reads a world file and rejects dangling exits.
func Load(path string) (*World, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s: %w", path, err)
	}
	var w World
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, fmt.Errorf("monde illisible: %w", err)
	}
	if err := w.index(); err != nil {
		return nil, err
	}
	return &w, nil
}

func (w *World) index() error {
	if w.Start == "" {
		return fmt.Errorf("monde: départ manquant")
	}
	w.byID = make(map[string]*Region, len(w.Regions))
	for i := range w.Regions {
		r := &w.Regions[i]
		if r.ID == "" || r.Name == "" {
			return fmt.Errorf("monde: région sans identifiant ou sans nom")
		}
		if _, ok := w.byID[r.ID]; ok {
			return fmt.Errorf("monde: identifiant en double")
		}
		if r.Danger < 1 {
			return fmt.Errorf("monde: danger invalide")
		}
		if len(r.Foes) == 0 {
			return fmt.Errorf("monde: aucun monstre dans une région")
		}
		w.byID[r.ID] = r
	}
	if _, ok := w.byID[w.Start]; !ok {
		return fmt.Errorf("monde: départ inconnu")
	}
	for _, r := range w.Regions {
		for _, ex := range r.Exits {
			if _, ok := w.byID[ex]; !ok {
				return fmt.Errorf("monde: sortie inconnue")
			}
		}
	}
	return nil
}

// Get returns the region with this id.
func (w *World) Get(id string) (*Region, bool) {
	r, ok := w.byID[id]
	return r, ok
}

// Find matches an id or a French name. Articles are optional.
func (w *World) Find(q string) (*Region, error) {
	want := Fold(q)
	if want == "" {
		return nil, fmt.Errorf("Région vide")
	}
	var hits []*Region
	for i := range w.Regions {
		r := &w.Regions[i]
		id := Fold(strings.ReplaceAll(r.ID, "-", " "))
		name := Fold(r.Name)
		short := strings.TrimPrefix(name, "la ")
		short = strings.TrimPrefix(short, "le ")
		short = strings.TrimPrefix(short, "l ")
		if want == id || want == name || want == short {
			hits = append(hits, r)
		}
	}
	if len(hits) == 0 {
		return nil, fmt.Errorf("Région inconnue: %s", q)
	}
	if len(hits) > 1 {
		return nil, fmt.Errorf("Région ambiguë: %s", q)
	}
	return hits[0], nil
}

// LeadsTo reports whether travel to id is an exit.
func (r *Region) LeadsTo(id string) bool {
	for _, ex := range r.Exits {
		if ex == id {
			return true
		}
	}
	return false
}

// Fold lowercases and strips accents so "Prés-au-Lard" matches "pres au lard".
func Fold(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'é', 'è', 'ê', 'ë':
			b.WriteRune('e')
		case 'à', 'â', 'ä':
			b.WriteRune('a')
		case 'ù', 'û', 'ü':
			b.WriteRune('u')
		case 'ô', 'ö':
			b.WriteRune('o')
		case 'î', 'ï':
			b.WriteRune('i')
		case 'ç':
			b.WriteRune('c')
		case '-', '\'', '’':
			b.WriteRune(' ')
		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
				b.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
