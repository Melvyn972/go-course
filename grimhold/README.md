# Grimhold

A terminal dungeon crawler. A broken Floo trunk in the Entrance Hall has stitched eight rooms together. You are a curse-breaker apprentice. Monster and spell statistics come from the [Open5e API](https://open5e.com) (WotC SRD, no API key). The program refuses any sheet whose `document__slug` is not `wotc-srd`.

Three creatures keep a local alias next to the SRD name: Giant Octopus (calmar géant), Giant Spider (acromantula), White Dragon Wyrmling (ironbelly jeune). `inspect` prints both.

## Build and run

Go 1.22 or later. Standard library only. Run from this directory so `world.json` is found.

```text
go run ./cmd/grimhold
go build -o grimhold.exe ./cmd/grimhold
grimhold.exe
grimhold.exe --continue
```

`--continue` loads `save.json`. If the file is missing, the program says so and starts a new character.

At creation you type a number. Each house, wand, and spell shows a text picture and one line on what it changes. A name still works (`magic missile` is accepted).

## Commands

`aide` is generated from the command map. The names the player sees are French. In a terminal the game is a menu: arrow keys move, Enter confirms. `aller`, `lancer`, `examiner`, and `bestiaire` open a second menu. A text picture is shown for the room, the creature, the spell, the house, and the wand whenever that thing is on screen. Piped input still accepts a command name or a leading number. The English names from the brief (`help`, `travel`, `fight`, …) still dispatch and are not listed.

| Command | Effect |
|---|---|
| `carte` | Every region, its danger, its exits. The vault is listed. |
| `aller <région>` | Move along an exit. French names and ids both work. A number picks an exit. |
| `regarder` | Description, danger, exits. |
| `explorer` | Roll `danger` times and keep the highest challenge rating. |
| `combattre` | In the menu, opens the fight: strike, spells, patronum, heal, or flee. Typed alone, one weapon exchange. |
| `fuir` | d20+2 against 12. Serdaigle adds +2. Failure is one free melee hit. |
| `reposer` | Full heal, spells, patronum and the potion returned. Once per entry. |
| `soigner` | 1d8+2 HP, once per rest. A foe still ripostes. |
| `fiche` | Sheet, including the HP bar. |
| `lancer <numéro>` | One of your two spells, once per rest. |
| `patronum` | Only against a creature whose type is Undead. Once per rest. |
| `examiner <monstre>` | French summary of the SRD sheet. |
| `bestiaire` / `bestiaire suivant` / `bestiaire précédent` | API pages, following `next` and `previous`. |
| `journal` | Killed, slept, driven off. |
| `sauver` / `charger` | `save.json`. |
| `quitter` | Leave. |

Empty input is ignored. An unknown command or a bad argument count prints a line and returns to the prompt.

## Character

Level 1 has 10 HP (12 for Poufsouffle) and attack bonus 2, plus the house and the wand.

| House | Effect |
|---|---|
| Gryffondor | +1 attack |
| Serdaigle | +2 on the flee roll |
| Poufsouffle | +2 max HP |
| Serpentard | Armor class and challenge rating are shown as soon as a foe appears |

| Wand | Dice | Extra |
|---|---|---|
| Phénix | 1d8 | |
| Dragon | 1d6 | +1 attack |
| Licorne | 1d4 | Episkey heals +2 |

Armor class is 12. Protego makes it 17 against the next melee riposte only.

You level up when total XP reaches level² × 50 (50, then 200, then 450 for level 4). Each level adds 4 max HP and 1 attack. XP by challenge rating: 0 → 10, 1/8 → 15, 1/4 → 25, 1/2 → 50, 1 → 100, 2 → 200.

A kill also rolls 1d4 galleons. There is nothing to buy.

## Spells

`cast` fetches `/v1/spells/<slug>/` and then applies a fixed rule. Incendio's DC is 12 because the sheet has no ability scores.

| Slug | Shown as | Rule |
|---|---|---|
| `burning-hands` | Incendio | 3d6 fire. A naked d20 against 12 halves it, rounded down. |
| `shield` | Protego | +5 AC against the melee riposte that follows. Not against a breath. |
| `cure-wounds` | Episkey | 1d8, +2 with the unicorn wand. Works out of combat. In combat it spends the turn. |
| `sleep` | Sommeil | 5d8. The foe sleeps if its current HP is less than or equal to the roll. No effect on Undead, or if the immunities contain `charmed`. |
| `magic-missile` | Projectiles magiques | 3×(1d4+1). It does not miss. |

Sleep and patronum grant the XP and no gold. The journal says `endormi` or `repoussé`. Only `tué` increments the kill count.

`patronum` is not an Open5e spell. On a non-undead the use is not spent.

## World

`explore` in a room of danger 2 rolls twice and keeps the worse foe. An ambush rolls once. Danger 1 never ambushes.

| Id | Name | Danger | What is there |
|---|---|---|---|
| `crossroads` | Le Hall | 1 | The trunk. Rats, an owl. Rest here is quiet. |
| `black-lake` | La vanne | 2 | Reef Shark, Giant Octopus. |
| `chamber` | La Chambre | 2 | Snakes. On entry, d20 against 12. Failure is -2 attack until the next rest. The skull is not a creature. |
| `forbidden-edge` | La lisière | 2 | Wolf, Giant Wolf Spider, Giant Spider. |
| `pres-au-lard` | Prés-au-Lard | 1 | A second quiet room, reached only through the edge. |
| `hospital-wing` | L'infirmerie | 2 | Shadow, Specter, Ghoul. `patronum` works here. Sommeil does not. |
| `trophy-room` | La salle des trophées | 2 | Flying Sword, Animated Armor, Mimic, Rug of Smothering. |
| `iron-vault` | Le caveau | 4 | White Dragon Wyrmling. `aller` is refused under level 4. |

The first time the wyrmling ripostes, it bites and then breathes: naked d20 against DC 12, 5d8 cold, half on a success. Later rounds recharge the breath on a 5 or 6. Those numbers are read from the action text. If the text cannot be read, the command fails. Poison riders and the shadow's Strength drain are not rolled: the sheet has no ability scores.

At 0 HP the wyrmling ends the curse. You can keep walking.

## Death and saving

At 0 HP you wake in the Hall at 1 HP. XP, gold, journal, spells, house, and wand stay. `save.json` is written only by `sauver`. Death, explore, fight, flee, cast, patronum, and an ambush do not touch the file.

## Cache

Responses are kept in a `map` for 5 minutes, guarded by a `sync.Mutex`. A goroutine purges expired entries every minute. Combat loads a creature with `GET /v1/monsters/<slug>/`, never with `?search=`.

## Ambush

In a room of danger 2 or more, with no foe already present, a goroutine ticks every 45 seconds. The prompt `select`s on stdin and on that tick. The trunk drops one foe from the current room. A line whose Enter was already pressed is dropped. A command typed after the spark is kept. `aller` resets the timer.

## Bonuses claimed: 12

- Spells, once per rest, fetched from `/v1/spells/`: **+4**
- Ambush goroutine and `select`: **+6**
- ANSI HP bar (house color) and danger colors (1 dim, 2 yellow, 4 red): **+2**

The vault, the houses, the wand, `patronum`, the chamber, and Prés-au-Lard are part of the world. They are not added on top of the 12.

`go test ./...` covers dice, Hit-clause parsing, level thresholds, death versus the save file, cache expiry on an injected clock, and Open5e decoding against `httptest`. Those tests do not hit the network and are not part of the 12.

## Sample session

Real run against the API. The owl, the roll of 21, the 2 galleons, and the skull's 11 are what the program printed. Typed commands are shown after the prompt; a pipe does not echo them into stdout.

```text
Trois choix. Tape le numéro.
...
Un numéro, ou une commande. aide pour la liste.
Le Hall
Le tronc est fendu sur les dalles. Des bancs retournés laissent passer des rats.
Danger 1, un seul tirage.
Sorties : La vanne, La lisière, L'infirmerie, La salle des trophées
████████████████████ 10/10 grimhold > carte
Le Hall — danger 1, un seul tirage ← ici
  sorties : La vanne, ...
Le caveau — danger 4, plusieurs tirages, un seul monstre
  sorties : La salle des trophées
  entrée : niveau 4
████████████████████ 10/10 grimhold > explorer
Devant toi : hibou.
████████████████████ 10/10 grimhold > combattre
Tu touches (21 contre CA 11). 2 dégâts.
hibou tombe. +10 PX, 2 galions.
████████████████████ 10/10 grimhold > combattre
Rien en face
████████████████████ 10/10 grimhold > aller la vanne
La vanne.
L'eau monte par une vanne coincée. Un siphon au fond tire vers le bas.
████████████████████ 10/10 grimhold > aller la chambre
La Chambre.
Un os plus long que la pièce gît au milieu, le crâne tourné vers l'échelle. L'eau du lac tombe encore du bec de la statue.
Le crâne est dans l'axe de l'échelle (11). -2 à l'attaque jusqu'au prochain repos.
████████████████████ 10/10 grimhold > aller la vanne
La vanne.
L'eau monte par une vanne coincée. Un siphon au fond tire vers le bas.
████████████████████ 10/10 grimhold > aller le hall
Le Hall.
Le tronc est fendu sur les dalles. Des bancs retournés laissent passer des rats.
████████████████████ 10/10 grimhold > reposer
PV au max. Sorts, patronum et soin rendus.
████████████████████ 10/10 grimhold > reposer
Tu t'es déjà reposé ici
████████████████████ 10/10 grimhold > aller le caveau
Aucun passage vers Le caveau
████████████████████ 10/10 grimhold > examiner acromantula
acromantula
Type bête, CA 14, PV 26, FP 1
Morsure : +5 pour toucher, ...
████████████████████ 10/10 grimhold > bestiaire
3207 fiches
A-mi-kuk  FP 7
...
bestiaire suivant
████████████████████ 10/10 grimhold > sauver
Sauvegarde écrite.
████████████████████ 10/10 grimhold > quitter
```
