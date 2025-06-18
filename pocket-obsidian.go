package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/fergalsomers/pocket-obsidian/csv"
	"github.com/fergalsomers/pocket-obsidian/page"

	flag "github.com/spf13/pflag"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	defaultOutpurDir         = "archive"
	defaultFailedCSVFilename = "failed.csv"
)

var (
	defaultTags  = []string{"clippings", "pocket"}
	outputDir    string   // Directory to write output files to, defaults to ./archive
	markRead     bool     // If true, mark articles as read in Pocket
	clippingTags []string // Default tags to add to all csv entries, defaults to clippings (per obsidian webclipper plugin)
	inputFile    string   // Arg 0 - the input CSV file containing Pocket records
	failedCSV    string
)

func init() {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	currentDir := filepath.Join(path, defaultOutpurDir)
	defaultCSVFile := filepath.Join(path, defaultFailedCSVFilename)
	flag.StringVarP(&outputDir, "output-dir", "o", currentDir, "Directory to write output files to defaults to ./archive")
	flag.BoolVarP(&markRead, "read", "r", false, "Mark articles as read in Pocket")
	flag.StringArrayVarP(&clippingTags, "tags", "t", defaultTags, "Default tags to add to all csv entries, defaults to clippings (per obsidian webclipper plugin)")
	flag.StringVarP(&failedCSV, "fail-csv", "f", defaultCSVFile, "Default tags to write failed entries to")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage of pocket-obsidian [input-csv-file]\n")
		flag.PrintDefaults()
	}

	flag.ErrHelp = errors.New("pocket-obsidian: Convert Mozilla Pocket exported CSV to Obsidian Markdown. See https://github.com/fergalsomers/pocket-obsidian")

	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "Error issing argument [input-csv-file]\n\n")
		flag.Usage()
		os.Exit(1)
	}
	inputFile = args[0]
}

type Result struct {
	Record []string
	Err    error
}

func main() {

	records, err := csv.ReadCSV(inputFile)
	if err != nil {
		log.Fatalf("Error reading CSV file: %v", err)
	}

	records = records[1:] // lose the header
	totalRecords := len(records)
	log.Printf("Read %d records from %s", totalRecords, inputFile)
	log.Printf("Writing records to %s", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	var wg sync.WaitGroup
	pc := mpb.New(mpb.WithWidth(80), mpb.WithWaitGroup(&wg))
	wg.Add(2)
	bar := pc.AddBar(int64(totalRecords),
		mpb.PrependDecorators(
			decor.Name("Processing:"),
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
		),
	)
	failedBar := pc.AddBar(int64(-1),
		mpb.PrependDecorators(
			decor.Name("Rejected:"),
			decor.CountersNoUnit("%d / %d"),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
		),
	)

	work := make(chan []string)
	results := make(chan Result)

	numWorkers := runtime.NumCPU()
	log.Printf("Number of processors: %d", numWorkers)

	// Start some workers to process the results
	c := page.NewContentRetriever()
	for i := 0; i < numWorkers; i++ {
		go func() {
			for {
				record, ok := <-work
				if !ok {
					return
				}
				_, err := page.RecordToClipping(c, outputDir, record, markRead, clippingTags)
				results <- Result{Record: record, Err: err}
			}
		}()
	}

	failedList := [][]string{}

	// Start the worker processing the results
	go func() {
		defer wg.Done()
		defer wg.Done()
		defer close(work)
		defer close(results)
		processResults(&failedList, len(records), results, bar, failedBar)

		failedTotal := int64(len(failedList))
		failedBar.SetTotal(failedTotal, true)

	}()

	// put recorcs on the work channel
	for _, record := range records {
		work <- record
	}

	pc.Wait() // the wg.Done above will cause this to stop blocking.

	if len(failedList) > 0 {
		log.Printf("Failed to retrieve %d entries", len(failedList))
		err := csv.WriteCSV(failedCSV,
			[]string{"title", "url", "time_added", "tags", "status", "error"},
			failedList)
		if err != nil {
			log.Fatal("Unable to write failedList %w", err)
		}
	}

}

// Used to process the results channel, we know how many results we need to get
func processResults(failedList *[][]string, numRecords int, results chan Result, bar *mpb.Bar, failedBar *mpb.Bar) {
	for i := numRecords; i > 0; i-- {
		r, ok := <-results
		if !ok {
			return
		}
		bar.Increment()
		if r.Err != nil {
			*failedList = append(*failedList, append(r.Record, r.Err.Error()))
			failedBar.Increment()
		}
	}
}
