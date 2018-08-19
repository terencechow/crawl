package parser_test

import (
	"flag"
	"fmt"
	"github.com/terencechow/crawl/parser"
	"os"
	"testing"
)

func resetFlagsForTesting() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestNoArguments(t *testing.T) {
	_, _, err := parser.GetCliArguments()
	if err == nil {
		t.Error("Expected error when no url provided")
	}
}

func TestInvalidURLArgument(t *testing.T) {
	invalidURLS := []string{
		"http//invalidscheme.com",
		"noscheme.com",
	}
	for _, invalidURL := range invalidURLS {
		resetFlagsForTesting()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{oldArgs[0], fmt.Sprintf("-url=%s", invalidURL), "-workers=3"}

		_, _, err := parser.GetCliArguments()
		if err == nil {
			t.Error("Expected error when invalid url provided")
		}
	}
}

func TestInvalidWorkersArgument(t *testing.T) {
	invalidWorkers := []int{
		0,
		-1,
		11,
	}
	for _, invalidWorker := range invalidWorkers {
		resetFlagsForTesting()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{oldArgs[0], "-url=https://www.google.com", fmt.Sprintf("-workers=%v", invalidWorker)}

		_, _, err := parser.GetCliArguments()
		if err == nil {
			t.Error("Expected error when value provided for workers")
		}
	}
}

func TestValidURLArgument(t *testing.T) {
	validURLS := []string{
		"https://www.google.com",
		"https://www.google.com?query=works",
		"https://www.google.com#fragment",
	}

	for _, validURL := range validURLS {

		resetFlagsForTesting()
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{oldArgs[0], fmt.Sprintf("-url=%s", validURL), "-workers=3"}

		url, workers, err := parser.GetCliArguments()
		if err != nil {
			t.Error("Expected error to be nil with a valid argument", err)
		}
		if url != "https://www.google.com" {
			t.Error("Expected url to be https://www.google.com")
		}
		if workers != 3 {
			t.Error("Expected number of workers to be 3")
		}
	}
}
