// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/GuanceCloud/mdcheck/check"
)

func checkMarkdown(dir string, checkSection bool, output io.Writer) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat markdown directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("markdown path is not a directory: %s", dir)
	}

	results, err := check.Check(
		check.WithMarkdownDir(dir),
		check.WithAutofix(false),
		check.WithCheckSection(checkSection),
	)
	if err != nil {
		return fmt.Errorf("check markdown: %w", err)
	}

	for _, result := range results {
		message := result.Err
		if message == "" {
			message = result.Warn
		}
		var writeErr error
		if result.Text == "" {
			_, writeErr = fmt.Fprintf(output, "%s: %s\n", result.Path, message)
		} else {
			_, writeErr = fmt.Fprintf(output, "%s: %s: %q\n", result.Path, message, result.Text)
		}
		if writeErr != nil {
			return fmt.Errorf("write markdown result: %w", writeErr)
		}
	}
	if len(results) > 0 {
		return fmt.Errorf("found %d markdown issue(s)", len(results))
	}

	return nil
}

func main() {
	dir := flag.String("dir", "", "markdown directory to check")
	checkSection := flag.Bool("check-section", true, "require section anchors")
	flag.Parse()

	if *dir == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := checkMarkdown(*dir, *checkSection, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
