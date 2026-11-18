package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/VoluteTech/grimhold/internal/game"
	"github.com/VoluteTech/grimhold/internal/open5e"
	"github.com/VoluteTech/grimhold/internal/repl"
	"github.com/VoluteTech/grimhold/internal/world"
)

func main() {
	os.Exit(run())
}

func run() int {
	args := os.Args[1:]
	for i, a := range args {
		if a == "--continuer" {
			args[i] = "--continue"
		}
	}
	os.Args = append([]string{os.Args[0]}, args...)
	cont := flag.Bool("continue", false, "charge la sauvegarde")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage : grimhold [--continuer]")
	}
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	defer repl.StopKeys()

	w, err := world.Load(findWorld())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	api := open5e.New(5 * time.Minute)
	go api.Reap(ctx, time.Minute)
	g := game.New(w, api)
	if *cont {
		err := g.Load(ctx, "save.json")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			if !errors.Is(err, game.ErrNoSave) {
				return 1
			}
		}
	}

	sc := bufio.NewScanner(os.Stdin)
	if !g.Ready() {
		house, wand, spells, err := repl.Create(ctx, sc)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		g.Start(house, wand, spells)
	}
	if err := repl.Run(ctx, sc, g); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func findWorld() string {
	candidates := []string{"world.json"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "world.json"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "world.json"
}
