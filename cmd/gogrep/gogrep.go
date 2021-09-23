package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/berquerant/gogrep"
)

const usage = `Usage of gogrep
  cat file | gogrep [flags] REGEX
  gogrep [flags] REGEX files...

Note:
The matched lines are not guaranteed to be in order in which they appear in the input.
Flags:`

func printUsage() {
	fmt.Fprintln(os.Stderr, usage)
	flag.PrintDefaults()
}

var (
	threads          = flag.Int("j", 4, "The number of grep workers. Positive number is valid.")
	resultBufferSize = flag.Int("b", 1000, "The size of grep result buffer. Positive number is valid.")
)

func main() {
	flag.Usage = printUsage
	flag.Parse()
	args := flag.Args()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	g := gogrep.New(
		gogrep.WithThreads(*threads),
		gogrep.WithResultBufferSize(*resultBufferSize),
	)
	if err := grep(ctx, g, args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		os.Exit(1)
	}
}

func grep(ctx context.Context, grepper gogrep.Grepper, args []string) error {
	switch len(args) {
	case 0:
		printUsage()
		return nil
	case 1:
		return grepStdin(ctx, grepper, args[0])
	case 2:
		return grepFile(ctx, grepper, args[0], args[1])
	default:
		return grepFiles(ctx, grepper, args[0], args[1:])
	}
}

func grepStdin(ctx context.Context, grepper gogrep.Grepper, regex string) error {
	resultC, err := grepper.Grep(ctx, regex, os.Stdin)
	if err != nil {
		return err
	}
	for r := range resultC {
		if err := r.Err(); err != nil {
			return err
		}
		fmt.Println(r.Text())
	}
	return nil
}

func grepFile(ctx context.Context, grepper gogrep.Grepper, regex, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	resultC, err := grepper.Grep(ctx, regex, f)
	if err != nil {
		return err
	}
	for r := range resultC {
		if err := r.Err(); err != nil {
			return err
		}
		fmt.Println(r.Text())
	}
	return nil
}

func grepFiles(ctx context.Context, grepper gogrep.Grepper, regex string, files []string) error {
	for _, file := range files {
		if err := func(file string) error {
			f, err := os.Open(file)
			if err != nil {
				return err
			}
			defer f.Close()
			resultC, err := grepper.Grep(ctx, regex, f)
			if err != nil {
				return err
			}
			for r := range resultC {
				if err := r.Err(); err != nil {
					return err
				}
				fmt.Printf("%s:%s\n", file, r.Text())
			}
			return nil
		}(file); err != nil {
			return err
		}
	}
	return nil
}
