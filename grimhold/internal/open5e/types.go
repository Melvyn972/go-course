package open5e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Monster is the subset of an Open5e v1 creature used in play.
type Monster struct {
	Slug                string     `json:"slug"`
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	ArmorClass          flexInt    `json:"armor_class"`
	HitPoints           int        `json:"hit_points"`
	ChallengeRating     CRString   `json:"challenge_rating"`
	Actions             []Action   `json:"actions"`
	ConditionImmunities immunities `json:"condition_immunities"`
	DocumentSlug        string     `json:"document__slug"`
}

// Action is one line of a stat block.
type Action struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
}

// Spell is the subset checked before a local casting rule runs.
type Spell struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	DocumentSlug string `json:"document__slug"`
}

// Page is one Open5e list response. Next and Previous are absolute URLs.
type Page struct {
	Count    int       `json:"count"`
	Next     *string   `json:"next"`
	Previous *string   `json:"previous"`
	Results  []Monster `json:"results"`
}

// flexInt accepts the v1 number and the later [{value:n}] shape.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '[' {
		var objs []struct {
			Value int `json:"value"`
		}
		if err := json.Unmarshal(b, &objs); err == nil && len(objs) > 0 {
			*f = flexInt(objs[0].Value)
			return nil
		}
		var nums []int
		if err := json.Unmarshal(b, &nums); err != nil {
			return fmt.Errorf("armor_class: %w", err)
		}
		if len(nums) > 0 {
			*f = flexInt(nums[0])
		}
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("armor_class: %w", err)
	}
	*f = flexInt(n)
	return nil
}

// CRString accepts "1/4" and a bare number.
type CRString string

func (c *CRString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*c = CRString(s)
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	*c = CRString(strconv.FormatFloat(f, 'f', -1, 64))
	return nil
}

// immunities accepts a string or a list of strings.
type immunities string

func (im *immunities) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*im = immunities(s)
		return nil
	}
	if b[0] == '[' {
		var arr []string
		if err := json.Unmarshal(b, &arr); err != nil {
			return err
		}
		*im = immunities(strings.Join(arr, ", "))
		return nil
	}
	return fmt.Errorf("immunités illisibles")
}

func (im immunities) String() string { return string(im) }
