package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-ci-time-causality/internal/causality"
)

func main() {
	flags := flag.NewFlagSet("time-causality", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	source := flags.String("source", "", "path to the .gooo source")
	contract := flags.String("contract", "", "path to the fixed denominator contract")
	corpus := flags.String("corpus", "", "path to the 12-case NDJSON corpus")
	fixture := flags.String("fixture", "", "path to the immutable public API fixture")
	out := flags.String("out", "", "caller-owned output directory")
	inventoryRoot := flags.String("inventory-root", ".", "checkout root used for exact input inventory")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	err := causality.Run(causality.RunOptions{
		SourcePath:    *source,
		ContractPath:  *contract,
		CorpusPath:    *corpus,
		FixturePath:   *fixture,
		OutputDir:     *out,
		InventoryRoot: *inventoryRoot,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("gooo-ci-time-causality: recorded six outputs")
}
