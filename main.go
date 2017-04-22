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
	VERSION        = "0.1"
	HEADER_OPTIONS = "options"
	HEADER_HEADERS = "headers"
	HEADER_ITEMS   = "items"
)

var (
	HEADERS = map[string]bool{
		HEADER_OPTIONS: true,
		HEADER_HEADERS: true,
		HEADER_ITEMS:   true,
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
	_, items := parse_csv()

	tmpl.Execute(os.Stdout, items)
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

func parse_csv() (options map[string]string, items []map[string]string) {
	const arg = 1

	logger.Printf("Opening input CSV at '%s'...", flag.Arg(arg))
	fh, err := os.Open(flag.Arg(arg))
	if err != nil {
		_die_on_err(err)
	}
	defer fh.Close()

	var item_headers []string
	var raw_items []map[string]string
	headers := make(map[string]string)
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
			if len(item_headers) == 0 {
				item_headers = record
				continue
			}

			item := make(map[string]string, len(item_headers))
			for idx, header := range item_headers {
				item[header] = record[idx]
			}

			raw_items = append(raw_items, item)
		case HEADER_HEADERS:
			headers[record[0]] = record[1]
		case HEADER_OPTIONS:
			options[record[0]] = record[1]
		default:
			_die_on_err(errors.New("Got garbage! Please run with `-v` flag."))
		}
	}

	logger.Printf("Parsed options: `%s`", options)
	logger.Printf("Parsed headers: `%s`", headers)
	logger.Printf("Parsed items: `%s`", raw_items)

	logger.Print("Merging headers into items...")
	for _, raw_item := range raw_items {
		item := make(map[string]string, len(item_headers)+len(headers))
		for k, v := range headers {
			item[k] = v
		}
		for k, v := range raw_item {
			item[k] = v
		}
		items = append(items, item)
	}

	return options, items
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
