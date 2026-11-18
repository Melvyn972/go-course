package open5e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const wolfJSON = `{
  "slug": "wolf",
  "name": "Wolf",
  "type": "Beast",
  "armor_class": [{"value": 13}],
  "hit_points": 11,
  "challenge_rating": "1/4",
  "condition_immunities": "",
  "document__slug": "wotc-srd",
  "actions": [{
    "name": "Bite",
    "desc": "Melee Weapon Attack: +4 to hit, reach 5 ft., one target. Hit: 7 (2d4 + 2) piercing damage.",
    "attack_bonus": 4
  }]
}`

func TestClientSRDAndCache(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch {
		case strings.Contains(r.URL.Path, "/monsters/wolf/"):
			_, _ = w.Write([]byte(wolfJSON))
		case strings.Contains(r.URL.Path, "/monsters/a-mi-kuk/"):
			_, _ = w.Write([]byte(`{"slug":"a-mi-kuk","name":"A-mi-kuk","type":"Aberration","armor_class":14,"hit_points":10,"challenge_rating":"7","document__slug":"tob2","condition_immunities":["paralyzed"]}`))
		case r.URL.Query().Get("page") == "2":
			_, _ = w.Write([]byte(`{"count":2,"next":null,"previous":"http://prev","results":[{"name":"Z","slug":"z","type":"Beast","armor_class":10,"hit_points":1,"challenge_rating":"1","document__slug":"wotc-srd"}]}`))
		default:
			next := "http://" + r.Host + r.URL.Path + "?page=2"
			_, _ = w.Write([]byte(`{"count":2,"next":"` + next + `","previous":null,"results":[{"name":"A","slug":"a","type":"Beast","armor_class":10,"hit_points":1,"challenge_rating":"0","document__slug":"wotc-srd"}]}`))
		}
	}))
	defer srv.Close()

	c := New(time.Minute)
	c.base = srv.URL
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.cache.now = func() time.Time { return now }

	m, err := c.Monster(context.Background(), "wolf")
	if err != nil {
		t.Fatal(err)
	}
	if int(m.ArmorClass) != 13 || m.Name != "Wolf" || string(m.ChallengeRating) != "1/4" {
		t.Fatalf("decoded %+v", m)
	}
	if _, err := c.Monster(context.Background(), "wolf"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("cache miss, hits %d", hits)
	}
	now = now.Add(2 * time.Minute)
	c.cache.purge()
	if _, err := c.Monster(context.Background(), "wolf"); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("expired entry still cached, hits %d", hits)
	}

	if _, err := c.Monster(context.Background(), "a-mi-kuk"); err == nil {
		t.Fatal("non-SRD sheet was accepted")
	}

	p, err := c.Page(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Next == nil || len(p.Results) != 1 {
		t.Fatalf("page %+v", p)
	}
	p2, err := c.Page(context.Background(), *p.Next)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Results[0].Slug != "z" {
		t.Fatalf("next page %+v", p2.Results)
	}
}
