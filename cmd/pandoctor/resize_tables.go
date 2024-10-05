package main

import (
	"bytes"
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/chrisfenner/pandoctor/pkg/gridtable"
)

var (
	matchColumns = flag.String("match_columns", "", "column headings to match (comma-separated, case-insensitive, MD formatting stripped)")
	newWidths    = flag.String("new_widths", "", "new widths for the column headings (comma-separated, base-10 integers)")
)

func validateResizeTablesArgs() error {
	matchCols := strings.Split(*matchColumns, ",")
	newWids := strings.Split(*newWidths, ",")
	if len(*matchColumns) == 0 {
		return fmt.Errorf("both --match_columns and --new_widths must be provided")
	}
	if len(matchCols) != len(newWids) {
		return fmt.Errorf("--match_columns must have the same number of (comma-separated) fields as --new_widths")
	}
	for _, wid := range newWids {
		if _, err := strconv.Atoi(wid); err != nil {
			return fmt.Errorf("--new_widths must be a comma-separated list of base-10 integers")
		}
	}
	return nil
}

func resizeTables(contents []byte) ([]byte, error) {
	tableRe := regexp.MustCompile("\\+[\\-\\+]+\n([|\\+].*\n)*\\+[\\-=\\+]+")
	return tableRe.ReplaceAllFunc(contents, resizeGridTable), nil
}

func resizeGridTable(contents []byte) []byte {
	config, cells, err := getTable(contents)
	if err != nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not read table: %v", err))
	}
	// We're not updating this table.
	if !matchTable(cells[0], config) {
		return contents
	}
	newTable, err := writeTable(*config, cells)
	if err != nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not write table: %v", err))
	}
	return []byte(newTable)
}

func getTable(contents []byte) (*gridtable.Config, [][]*gridtable.Cell, error) {
	r, err := gridtable.NewReader(bytes.NewReader(contents))
	if err != nil {
		return nil, nil, fmt.Errorf("could not initialize table reader: %v", err)
	}
	var cells [][]*gridtable.Cell
	for row, err := range r.Read() {
		if err != nil {
			return nil, nil, fmt.Errorf("could not read table row: %v", err)
		}
		cells = append(cells, row)
	}
	config, err := r.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not read table config: %v", err)
	}
	return config, cells, nil
}

func matchTable(firstRow []*gridtable.Cell, config *gridtable.Config) bool {
	var headings []string
	for _, cell := range firstRow {
		// Current version doesn't support resizing tables with spans in the header.
		if cell == nil {
			return false
		}
		headings = append(headings, cell.Text)
	}
	// Compare all the headings to the flag passed in.
	matchCols := strings.Split(*matchColumns, ",")
	if len(headings) != len(matchCols) {
		return false
	}
	for i, heading := range headings {
		// Trim Markdown formatting and compare case-insensitive.
		heading = strings.ReplaceAll(heading, "*", "")
		heading = strings.ReplaceAll(heading, "_", "")
		heading = strings.ReplaceAll(heading, "`", "")
		if !strings.EqualFold(heading, matchCols[i]) {
			return false
		}
	}
	// Update the config based on the passed-in widths.
	for i, width := range strings.Split(*newWidths, ",") {
		w, err := strconv.Atoi(width)
		if err != nil {
			panic("unexpectedly failed to parse width from new_widths")
		}
		config.Columns[i].Width = w
	}
	return true
}

func writeTable(config gridtable.Config, cells [][]*gridtable.Cell) (string, error) {
	w, err := gridtable.NewWriter(config)
	if err != nil {
		return "", fmt.Errorf("could not initialize table writer: %v", err)
	}
	for _, row := range cells {
		for i, cell := range row {
			if cell == nil {
				continue
			}
			w.WriteColumn(i, *cell)
		}
		w.NextRow()
	}
	return w.String()
}
