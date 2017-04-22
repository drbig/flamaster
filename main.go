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
)

type (
	Item    map[string]string
	Headers map[string]string
	Options map[string]string
	Data    struct {
		Headers
		Item
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

	tmpl := parse_template()
	options, headers, items := parse_csv()

	if len(options) > 0 {
		logger.Print("Merging options...")
		for k, v := range options {
			if err := flag.Set(k, v); err != nil {
				_die_on_err(err)
			}
		}
	}

	if flagVerbose {
		flag.VisitAll(func(f *flag.Flag) {
			logger.Printf(
				"Name: `%s`, Value: `%s`, Default: `%s`",
				f.Name, f.Value, f.DefValue,
			)
		})
	}

	if flagSingleOutput {
		logger.Print("Running all items once...")
		tmpl.Execute(
			os.Stdout,
			Data{Headers: headers, Items: items},
		)
		return
	}

	logger.Print("Running template per item...")
	for _, item := range items {
		tmpl.Execute(
			os.Stdout,
			Data{Headers: headers, Item: item},
		)
	}
}

func parse_template() (tmpl *template.Template) {
	const arg = 0
	var err error

	logger.Printf("Parsing template at '%s'...", flag.Arg(arg))
	tmpl, err = template.ParseFiles(flag.Arg(0))
	if err != nil {
		_die_on_err(err)
	}

	return tmpl
}

func parse_csv() (options Options, headers Headers, items []Item) {
	const arg = 1
	options = make(Options)
	headers = make(Headers)

	logger.Printf("Opening input CSV at '%s'...", flag.Arg(arg))
	fh, err := os.Open(flag.Arg(arg))
	if err != nil {
		_die_on_err(err)
	}
	defer fh.Close()

	var item_headers []string
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

			items = append(items, item)
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
	logger.Printf("Parsed items: `%s`", items)

	return options, headers, items
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
	fmt.Fprintln(os.Stderr, "Please run with -v to see where this happened.")
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
