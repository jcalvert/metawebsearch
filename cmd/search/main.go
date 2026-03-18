package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jcalvert/metawebsearch"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: search <query>\n")
		os.Exit(1)
	}

	query := strings.Join(os.Args[1:], " ")

	client, err := metawebsearch.NewClient(metawebsearch.ClientOpts{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
		os.Exit(1)
	}

	ms := metawebsearch.MultiSearch{
		Client:  client,
		Engines: metawebsearch.AllEngines(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sr, err := ms.Search(ctx, query, metawebsearch.SearchOpts{MaxResults: 10})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Search error: %v\n", err)
		os.Exit(1)
	}

	out := struct {
		Query   string                    `json:"query"`
		Results []metawebsearch.Result    `json:"results"`
		Errors  map[string]string         `json:"errors,omitempty"`
	}{
		Query:   query,
		Results: sr.Results,
		Errors:  make(map[string]string),
	}

	for name, e := range sr.Errors {
		out.Errors[name] = e.Error()
	}
	if len(out.Errors) == 0 {
		out.Errors = nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
