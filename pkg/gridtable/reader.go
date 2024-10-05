package gridtable

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
)

// Reader is an object that can be used to read in a grid table.
type Reader struct {
	scanner *bufio.Scanner
	config  Config
	numRows int
	done    bool
}

// NewReader instantiates a new Reader that reads table rows from an underlying io.Reader.
func NewReader(r io.Reader) (*Reader, error) {
	scanner := bufio.NewScanner(r)
	// Go ahead and read in the top line to get things started
	if !scanner.Scan() {
		return nil, fmt.Errorf("%w: table needs to contain at least one line of text", ErrMalformedTable)
	}
	topLine := scanner.Text()
	cols, isHdr, err := validateSeparator(topLine)
	if err != nil {
		return nil, err
	}
	if isHdr {
		// Not OK at this time.
		return nil, fmt.Errorf("%w: table cannot begin with '=' symbols", ErrMalformedTable)
	}

	return &Reader{
		scanner: scanner,
		// We don't know how many rows are in the header at this time. That's OK, we'll find out later.
		config: Config{
			Columns: cols,
		},
	}, nil
}

// validateSeparator returns the column-spec array described by the horizontal separator, or an error if the line is not a separator.
// Note that at this time, row-spans are not supported.
func validateSeparator(line string) (cols []ColumnSpec, isHeader bool, err error) {
	headerDecided := false
	isHdr := false
	if len(line) == 0 {
		return nil, false, fmt.Errorf("%w: line of table cannot be empty", ErrMalformedTable)
	}
	if line[0] != '+' || line[len(line)-1] != '+' {
		return nil, false, fmt.Errorf("%w: separator must start and end with '+'", ErrMalformedTable)
	}
	line = line[1 : len(line)-1]
	for i, col := range strings.Split(line, "+") {
		colWidth := len(col)
		if colWidth < minColumnWidth {
			return nil, false, fmt.Errorf("%w: column %v too narrow at %v characters wide", ErrMalformedTable, i, colWidth)
		}
		// Check for funny business. We expect every character in col to be a - or a = (and all the same)
		for _, char := range col {
			switch {
			case !headerDecided && char == '-':
				// OK, and remember for next time.
				headerDecided = true
				isHeader = false
			case !headerDecided && char == '=':
				// OK, and remember for next time.
				headerDecided = true
				isHeader = true
			case headerDecided && !isHdr && char == '-':
				// OK
			case headerDecided && isHdr && char == '=':
				// OK
			default:
				return nil, false, fmt.Errorf("%w: unexpected character %q in separator line", ErrMalformedTable, char)
			}
		}
		cols = append(cols, ColumnSpec{Width: len(col)})
	}
	if len(cols) == 0 {
		return nil, false, fmt.Errorf("%w: table needs to contain at least one column", ErrMalformedTable)
	}
	return cols, isHdr, nil
}

// scanToNextSeparator reads to the next horizontal separator, returning the raw content in between and an indicator
// of whether the header separator was encountered. It validates that the separator it found agrees with the passed-in
// Config.
func scanToNextSeparator(config Config, scanner *bufio.Scanner) (rawContents [][]rune, isHeader bool, err error) {
	var result [][]rune

	for scanner.Scan() {
		cols, isHdr, err := validateSeparator(scanner.Text())
		if err != nil {
			// Assume it's jut not a separator.
			result = append(result, []rune(scanner.Text()))
			continue
		}
		// Found a separator. Check that the columns agree.
		if len(cols) != len(config.Columns) {
			return nil, false, fmt.Errorf("%w: number of columns appeared to change midway through this table", ErrMalformedTable)
		}
		for i, col := range cols {
			if col.Width != config.Columns[i].Width {
				return nil, false, fmt.Errorf("%w: width of column %v appeared to change midway through this table", ErrMalformedTable, i)
			}
		}
		return result, isHdr, nil
	}
	// Special case: the table string might contain an empty line. If so, just return io.EOF and stop scanning.
	if len(scanner.Text()) == 0 {
		return nil, false, io.EOF
	}
	return nil, false, fmt.Errorf("%w: found content past the end of the table", ErrMalformedTable)
}

// cellsFromContent converts raw content into an array of cells. It uses the column configuration to determine if there
// are any column spans. Shadowed cells are represented in the array as nil values.
// Note that row spans are not supported at this time.
func cellsFromContent(config Config, lines [][]rune) ([]*Cell, error) {
	// Basic validation
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: each row needs to have at least one line of text", ErrMalformedTable)
	}
	expectedLineLen := 1
	for _, col := range config.Columns {
		expectedLineLen += col.Width + 1
	}
	for _, line := range lines {
		if len(line) != expectedLineLen {
			return nil, fmt.Errorf("%w: each line of text needs to have the same width", ErrMalformedTable)
		}
		if line[0] != '|' || line[len(line)-1] != '|' {
			return nil, fmt.Errorf("%w: each line of text needs to begin and end with a '|'", ErrMalformedTable)
		}
	}
	result := make([]*Cell, len(config.Columns))
	// Check for spans and initialize the result array
	currentCellIndex := 0
	span := 0
	dx := 0
	for _, col := range config.Columns {
		dx += 1 + col.Width
		if lines[0][dx] == '|' {
			// Found the right edge of whatever the current cell is.
			// That means we can initialize the current Cell.
			result[currentCellIndex] = &Cell{
				ColSpan: span,
			}
			currentCellIndex += 1 + span
			span = 0
		} else {
			// Current cell spans. Keep going.
			span++
		}
	}
	// Populate the cells' text based on the raw lines
	x := 1
	for i, cell := range result {
		if cell == nil {
			continue
		}
		dx := 0
		for j := i; j <= i+cell.ColSpan; j++ {
			dx += config.Columns[i].Width
		}
		// Don't forget the additional space made by the missing '|'s, for spanning cells.
		dx += cell.ColSpan
		cell.Text = readCellContents(lines, x, x+dx)
		x += dx + 1
	}
	return result, nil
}

// Read() returns an iterator over rows that can be ranged over using the range function.
// A nil entry in the result indicates a shadowed cell.
// It returns EndOfTable, nil when there are no more rows to read.
func (r *Reader) Read() iter.Seq2[[]*Cell, error] {
	return func(yield func([]*Cell, error) bool) {
		if r.done {
			yield(nil, io.EOF)
			return
		}
		for {
			// Look for the next separator.
			content, isHdr, err := scanToNextSeparator(r.config, r.scanner)
			//
			// Check for EOF and signal if needed.
			if errors.Is(err, io.EOF) {
				r.done = true
				return
			}
			// Check for other errors.
			if err != nil {
				yield(nil, err)
				return
			}
			// Convert the content to cells and check for errors.
			cells, err := cellsFromContent(r.config, content)
			if err != nil {
				yield(nil, err)
				return
			}
			r.numRows++
			if isHdr {
				r.config.NumHeaderRows = r.numRows
			}
			// Yield the current row of cells.
			if !yield(cells, nil) {
				return
			}
		}
	}
}

// GetConfig can be used to read the config detected on the table.
func (r *Reader) GetConfig() (*Config, error) {
	if !r.done {
		return nil, ErrReaderNotDone
	}
	return &r.config, nil
}
