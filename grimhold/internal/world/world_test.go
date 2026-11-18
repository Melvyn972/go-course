package world

import (
	"path/filepath"
	"testing"
)

func TestRepoWorld(t *testing.T) {
	w, err := Load(filepath.Join("..", "..", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	if w.Start != "crossroads" {
		t.Fatalf("start %s", w.Start)
	}
	if len(w.Regions) != 8 {
		t.Fatalf("regions %d", len(w.Regions))
	}
	if _, err := w.Find("chambre"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Find("Prés-au-Lard"); err != nil {
		t.Fatal(err)
	}
	lake, ok := w.Get("black-lake")
	if !ok || !lake.LeadsTo("chamber") {
		t.Fatal("la vanne doit mener à la chambre")
	}
	edge, ok := w.Get("forbidden-edge")
	if !ok || !edge.LeadsTo("pres-au-lard") {
		t.Fatal("la lisière doit mener à Prés-au-Lard")
	}
	hall, ok := w.Get("crossroads")
	if !ok || hall.LeadsTo("iron-vault") || hall.LeadsTo("chamber") {
		t.Fatal("le hall ne mène ni au caveau ni à la chambre")
	}
}
