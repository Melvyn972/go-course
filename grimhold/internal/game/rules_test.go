package game

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/VoluteTech/grimhold/internal/world"
)

func TestHitTerms(t *testing.T) {
	cases := []struct {
		name string
		desc string
		want []string
	}{
		{
			name: "wyrmling bite plus cold",
			desc: "Melee Weapon Attack: +4 to hit, reach 5 ft., one target. Hit: 7 (1d10 + 2) piercing damage plus 2 (1d4) cold damage.",
			want: []string{"1d10+2", "1d4"},
		},
		{
			name: "spider poison stays out",
			desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one creature. Hit: 7 (1d8 + 3) piercing damage, and the target must make a DC 11 Constitution saving throw, taking 9 (2d8) poison damage on a failed save, or half as much damage on a successful one.",
			want: []string{"1d8+3"},
		},
		{
			name: "snake flat piercing",
			desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one target. Hit: 1 piercing damage, and the target must make a DC 10 Constitution saving throw, taking 5 (2d4) poison damage on a failed save, or half as much damage on a successful one.",
			want: []string{"1"},
		},
		{
			name: "rug smother",
			desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one Medium or smaller creature. Hit: The creature is grappled (escape DC 13). Until this grapple ends, the target is restrained, blinded, and at risk of suffocating, and the rug can't smother another target. In addition, at the start of each of the target's turns, the target takes 10 (2d6 + 3) bludgeoning damage.",
			want: []string{"2d6+3"},
		},
		{
			name: "swarm half-hp clause ignored",
			desc: "Melee Weapon Attack: +6 to hit, reach 0 ft., one creature in the swarm's space. Hit: 7 (2d6) piercing damage, or 3 (1d6) piercing damage if the swarm has half of its hit points or fewer. The target must make a DC 10 Constitution saving throw, taking 14 (4d6) poison damage on a failed save, or half as much damage on a successful one.",
			want: []string{"2d6"},
		},
		{
			name: "shadow drain ignored",
			desc: "Melee Weapon Attack: +4 to hit, reach 5 ft., one creature. Hit: 9 (2d6 + 2) necrotic damage, and the target's Strength score is reduced by 1d4.",
			want: []string{"2d6+2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := HitTerms(tc.desc)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v want %v", got, tc.want)
				}
			}
		})
	}
}

func TestBestStrike(t *testing.T) {
	ghoul := []Action{
		{Name: "Bite", Desc: "Melee Weapon Attack: +2 to hit, reach 5 ft., one creature. Hit: 9 (2d6 + 2) piercing damage.", AttackBonus: 2},
		{Name: "Claws", Desc: "Melee Weapon Attack: +4 to hit, reach 5 ft., one target. Hit: 7 (2d4 + 2) slashing damage. If the target is a creature other than an elf or undead, it must succeed on a DC 10 Constitution saving throw or be paralyzed for 1 minute.", AttackBonus: 0},
	}
	s, err := BestStrike(ghoul)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Claws" || s.Bonus != 4 || len(s.Parts) != 1 || s.Parts[0] != "2d4+2" {
		t.Fatalf("%+v", s)
	}

	spider := []Action{
		{Name: "Bite", Desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one creature. Hit: 7 (1d8 + 3) piercing damage, and the target must make a DC 11 Constitution saving throw, taking 9 (2d8) poison damage on a failed save.", AttackBonus: 5},
		{Name: "Web (Recharge 5-6)", Desc: "Ranged Weapon Attack: +5 to hit, range 30/60 ft., one creature. Hit: The target is restrained by webbing.", AttackBonus: 5},
	}
	s, err = BestStrike(spider)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Bite" {
		t.Fatalf("picked %s", s.Name)
	}

	mimic := []Action{
		{Name: "Pseudopod", Desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one target. Hit: 7 (1d8 + 3) bludgeoning damage.", AttackBonus: 5},
		{Name: "Bite", Desc: "Melee Weapon Attack: +5 to hit, reach 5 ft., one target. Hit: 7 (1d8 + 3) piercing damage plus 4 (1d8) acid damage.", AttackBonus: 5},
	}
	s, err = BestStrike(mimic)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Bite" || len(s.Parts) != 2 {
		t.Fatalf("%+v", s)
	}
}

func TestParseBreath(t *testing.T) {
	actions := []Action{
		{Name: "Bite", Desc: "Melee Weapon Attack: +4 to hit. Hit: 7 (1d10 + 2) piercing damage plus 2 (1d4) cold damage."},
		{Name: "Cold Breath (Recharge 5-6)", Desc: "The dragon exhales an icy blast of hail in a 15-foot cone. Each creature in that area must make a DC 12 Constitution saving throw, taking 22 (5d8) cold damage on a failed save, or half as much damage on a successful one."},
	}
	br, err := ParseBreath(actions)
	if err != nil {
		t.Fatal(err)
	}
	if br.Dice != "5d8" || br.DC != 12 || br.Recharge != 5 {
		t.Fatalf("%+v", br)
	}
}

func TestRoll(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	n, err := Roll(rng, "1d1")
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	n, err = Roll(rng, "2d1+3")
	if err != nil || n != 5 {
		t.Fatal(n, err)
	}
	n, err = Roll(rng, "4")
	if err != nil || n != 4 {
		t.Fatal(n, err)
	}
	if _, err := Roll(rng, "nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLevelThresholds(t *testing.T) {
	c := NewCharacter("gryffondor", "phenix", []string{"sleep", "shield"})
	if c.MaxHP != 10 || c.AttackBonus() != 3 {
		t.Fatalf("start hp %d atk %d", c.MaxHP, c.AttackBonus())
	}
	c.gainXP(49)
	if c.Level != 1 {
		t.Fatal(c.Level)
	}
	c.gainXP(1)
	if c.Level != 2 || c.MaxHP != 14 {
		t.Fatalf("lv %d hp %d", c.Level, c.MaxHP)
	}
	c.gainXP(150)
	if c.Level != 3 {
		t.Fatal(c.Level)
	}
	c.gainXP(250)
	if c.Level != 4 || c.MaxHP != 22 || c.XP != 450 {
		t.Fatalf("lv %d hp %d xp %d", c.Level, c.MaxHP, c.XP)
	}
}

func TestHufflepuffAndGaze(t *testing.T) {
	c := NewCharacter("poufsouffle", "licorne", []string{"cure-wounds", "shield"})
	if c.MaxHP != 12 {
		t.Fatal(c.MaxHP)
	}
	base := c.AttackBonus()
	c.Gaze = true
	if c.AttackBonus() != base-2 {
		t.Fatalf("gaze atk %d base %d", c.AttackBonus(), base)
	}
}

func TestSleepImmune(t *testing.T) {
	g := &Game{Foe: &Foe{Type: "Undead"}}
	if !g.sleepImmune() {
		t.Fatal("undead")
	}
	g.Foe = &Foe{Type: "Construct", Immunities: "blinded, charmed, deafened"}
	if !g.sleepImmune() {
		t.Fatal("charmed")
	}
	g.Foe = &Foe{Type: "Beast"}
	if g.sleepImmune() {
		t.Fatal("beast")
	}
}

func TestDeathLeavesSaveUntouched(t *testing.T) {
	w, err := world.Load(filepath.Join("..", "..", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := New(w, nil)
	g.Start("serdaigle", "dragon", []string{"shield", "sleep"})
	g.Loc = "black-lake"
	g.Char.HP = 4
	g.Char.Gold = 3
	g.Foe = &Foe{Slug: "wolf", Name: "Wolf", HP: 11}
	path := filepath.Join(t.TempDir(), "save.json")
	if err := g.Save(path); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	msg, died := g.hurt(9)
	if !died || g.Loc != "crossroads" || g.Char.HP != 1 || g.Foe != nil || g.Char.Gold != 3 {
		t.Fatalf("death state loc=%s hp=%d foe=%v gold=%d msg=%q", g.Loc, g.Char.HP, g.Foe, g.Char.Gold, msg)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("death rewrote save.json")
	}
}

func TestHealOncePerRest(t *testing.T) {
	g := &Game{rng: rand.New(rand.NewPCG(1, 2))}
	g.Char = NewCharacter("gryffondor", "phenix", []string{"shield", "sleep"})
	g.Char.HP = 1
	if _, err := g.Heal(); err != nil {
		t.Fatal(err)
	}
	if g.Char.HP < 4 || g.Char.HP > g.Char.MaxHP {
		t.Fatalf("hp %d", g.Char.HP)
	}
	if _, err := g.Heal(); err == nil {
		t.Fatal("second heal should fail")
	}
	g.Char.recover()
	if g.Char.SoinSpent || g.Char.HP != g.Char.MaxHP {
		t.Fatalf("rest hp %d spent %v", g.Char.HP, g.Char.SoinSpent)
	}
	if _, err := g.Heal(); err == nil {
		t.Fatal("full heal should fail")
	}
}

func TestXPTable(t *testing.T) {
	if xpFor("0") != 10 || xpFor("1/8") != 15 || xpFor("1/4") != 25 || xpFor("1/2") != 50 || xpFor("1") != 100 || xpFor("2") != 200 {
		t.Fatal("xp table")
	}
}
