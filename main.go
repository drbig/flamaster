package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"
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
	tmpl                   *template.Template
)

type (
	Item             map[string]string
	MultiItemPayload struct {
		Items []Item
	}
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
		&flagSingleOutput, "os",
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

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "You have to specify a template and a CSV.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	parse_template()
	items := parse_csv()

	logger.Print(items)

	payload := MultiItemPayload{Items: items}
	tmpl.Execute(os.Stdout, payload)
}

func parse_template() {
	const arg = 0
	var err error

	logger.Printf("Parsing template at '%s'...", flag.Arg(arg))
	tmpl, err = template.ParseFiles(flag.Arg(0))
	if err != nil {
		_die_on_err(err)
	}
}

func parse_csv() (items []Item) {
	const arg = 1

	logger.Printf("Opening input CSV at '%s'...", flag.Arg(arg))
	fh, err := os.Open(flag.Arg(arg))
	if err != nil {
		_die_on_err(err)
	}
	defer fh.Close()

	var headers []string
	section := ""

	logger.Print("Parsing CSV file...")
	csv := csv.NewReader(bufio.NewReader(fh))
	for {
		record, done := _read_csv_record(csv)
		if done {
			break
		}
		logger.Printf("At row: `%s`", record)

		if section != "" && record[0] == "" {
			section = ""
			continue
		}

		if record[0] == "" {
			continue
		}

		if section == "" && HEADERS[record[0]] {
			section = record[0]
			continue
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
		default:
			_die_on_err(errors.New("Got garbage! Please run with `-v` flag."))
		}
	}

	return items
}

func _read_csv_record(r *csv.Reader) (record []string, done bool) {
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
