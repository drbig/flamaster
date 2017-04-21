package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	VERSION      = "0.1"
	HEADER_ITEMS = "items"
)

var (
	HEADERS = map[string]bool{
		HEADER_ITEMS: true,
	}
)

var (
	flagVerbose            bool
	flagSingleOutput       bool
	flagOutputFileTemplate string
	flagOutputRoot         string
	logger                 *log.Logger
)

type (
	Item map[string]string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s (options...) file.csv
flamaster v%s, see LICENSE.txt

`,
			os.Args[0], VERSION,
		)
		flag.PrintDefaults()
	}

	flag.BoolVar(&flagVerbose, "v", false, "be very verbose")
	flag.BoolVar(
		&flagSingleOutput, "s",
		false,
		"process all items within single template run",
	)
	flag.StringVar(
		&flagOutputFileTemplate, "ot",
		"",
		"template string for generating output file names. "+
			"If empty will just print to stdout.",
	)
	flag.StringVar(
		&flagOutputRoot, "or",
		func() string { path, _ := os.Getwd(); return path }(),
		"path to where to save output files",
	)
}

func main() {
	flag.Parse()
	if flagVerbose {
		logger = log.New(os.Stderr, "", log.Lshortfile|log.Ltime)
		logger.Print("Verbose logging enabled.")
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "No input csv file name specified.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	logger.Printf("Opening input CSV at '%s'...", flag.Arg(0))
	fh, err := os.Open(flag.Arg(0))
	if err != nil {
		_die_on_err(err)
	}
	defer fh.Close()

	// Parse CSV
	var headers []string
	var items []Item
	section := ""

	logger.Print("Parsing CSV file...")
	csv := csv.NewReader(bufio.NewReader(fh))
	for {
		record, done := read_record(csv)
		if done {
			break
		}
		logger.Printf("At row: `%s`", record)

		if section == "" && HEADERS[record[0]] {
			section = record[0]
			continue
		}

		if record[0] == "" {
			if section != "" {
				section = ""
				continue
			}
		}

		switch section {
		case HEADER_ITEMS:
			if len(headers) == 0 {
				headers = record
				continue
			}

			item := make(Item, len(headers))
			for idx, header := range headers {
				item[header] = record[idx]
			}

			items = append(items, item)
		}
	}
}

func read_record(r *csv.Reader) (record []string, done bool) {
	record, err := r.Read()

	if err == io.EOF {
		return record, true
	} else if err != nil {
		_die_on_err(err)
	}

	return record, false
}

func _die_on_err(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}