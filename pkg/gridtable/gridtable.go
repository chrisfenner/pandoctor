// Package gridtable implements a library for printing grid tables.
package gridtable

import (
	"errors"
	"fmt"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

var (
	// ErrColumnIndexOutOfRange indicates that an invalid column index was referenced.
	ErrColumnIndexOutOfRange = errors.New("column index out of range")
	// ErrShadowedCell indicates that a "shadowed" cell (one hidden by an already-spanned cell) was referenced.
	ErrShadowedCell = errors.New("wrote to shadowed cell")
	// ErrNegativeSpan indicates that a cell with a negative span was written.
	ErrNegativeSpan = errors.New("negative span")
	// ErrOverlappingSpans indicates that two different spans overlapped.
	ErrOverlappingSpans = errors.New("overlapping spans")
	// ErrSpanBeyondHeader indicates that a cell in the header spanned past the end of the header.
	ErrSpanBeyondHeader = errors.New("span extended beyond header")
	// ErrInvalidColumnSpec indicates that a column spec was invalid.
	ErrInvalidColumnSpec = errors.New("invalid column spec")
	// ErrBadWrap indicates that text could not be wrapped to fit into its column.
	ErrBadWrap = errors.New("text could not be wrapped")
	// ErrMalformedTable indicates that the grid table was malformed.
	ErrMalformedTable = errors.New("malformed grid table")
	// ErrReaderNotDone indicates that the requested operation requires the reader to have completely consumed the table already,
	// and it hasn't.
	ErrReaderNotDone = errors.New("reader not done reading table")
)

const (
	// A column with room for one character in it with padding on both sides.
	minColumnWidth = 3
)

// A ColumnSpec describes the parameters of a column.
type ColumnSpec struct {
	// Width of the column in number of characters (not counting the separators).
	Width int
}

// Cell is the contents to write into the cell of a table.
type Cell struct {
	// The text to put into the cell.
	Text string
	// The number of additional rows this cell spans. 0 = no span.
	RowSpan int
	// The number of additional columns this cell spans. 0 = no span.
	ColSpan int
}

// String implements Stringer.
func (c *Cell) String() string {
	if c == nil {
		return "[shadowed cell]"
	}
	spanStr := ""
	if c.ColSpan > 0 {
		spanStr += fmt.Sprintf("colspan=%v", c.ColSpan)
	}
	if c.RowSpan > 0 {
		spanStr += fmt.Sprintf("rowspan=%v", c.RowSpan)
	}
	return fmt.Sprintf("%v[%v]", spanStr, c.Text)
}

func calculateTableWidth(cols []ColumnSpec) int {
	result := 1 // Left pipe.
	for _, col := range cols {
		result += col.Width + 1 // This column's width plus the pipe on the right of it.
	}
	return result
}

func calculateTableHeight(rows []int) int {
	result := 1 // Top pipe.
	for _, row := range rows {
		result += row + 1 // This row's height plus the pipe below it.
	}
	return result
}

func cellHeight(row int, cell *Cell, rowHeights []int) int {
	result := rowHeights[row]
	for i := row + 1; i <= row+cell.RowSpan; i++ {
		result += rowHeights[i] + 1 // The pipe.
	}
	return result
}

func cellWidth(column int, cell *Cell, colSpec []ColumnSpec) int {
	result := colSpec[column].Width
	for j := column + 1; j <= column+cell.ColSpan; j++ {
		result += colSpec[j].Width + 1 // We get to reclaim the space where the | would go.
	}
	return result
}

func lines(column int, cell *Cell, colSpec []ColumnSpec) ([]string, error) {
	limit := cellWidth(column, cell, colSpec) - 2 // leave room for spaces on both sides of the content
	ww := wordwrap.NewWriter(limit)
	ww.KeepNewlines = true

	ww.Write([]byte(cell.Text))
	if err := ww.Close(); err != nil {
		return nil, fmt.Errorf("%w: text in column %d could not be wrapped", ErrBadWrap, column)
	}
	wrapped := ww.String()
	lines := strings.Split(wrapped, "\n")
	for _, line := range lines {
		if len(line) > limit {
			return nil, fmt.Errorf("%w: text in column %d could not be wrapped", ErrBadWrap, column)
		}
	}

	return lines, nil
}

func calculateCellHeight(column int, cell *Cell, colSpec []ColumnSpec) (int, error) {
	lines, err := lines(column, cell, colSpec)
	if err != nil {
		return 0, err
	}
	return len(lines), nil
}

func drawCellContents(array [][]rune, x int, y int, row, column int, cell *Cell, colSpec []ColumnSpec, rowHeights []int) error {
	// Start by erasing the interior of the cell,
	width := cellWidth(column, cell, colSpec)
	height := cellHeight(row, cell, rowHeights)
	for dx := 0; dx < width; dx++ {
		for dy := 0; dy < height; dy++ {
			array[x+dx][y+dy] = ' '
		}
	}

	lines, err := lines(column, cell, colSpec)
	if err != nil {
		return err
	}
	for dy, line := range lines {
		for dx, r := range line {
			// Draw the cell contents one space to the right of the left |.
			array[x+1+dx][y+dy] = r
		}
	}
	return nil
}

// readCellContents performs a rectangular full-height selection of the text in the range [start, end).
// It trims all extraneous whitespace, but preserves double-newlines.
func readCellContents(lines [][]rune, start int, end int) string {
	var result strings.Builder
	for i := range lines {
		line := strings.TrimSpace(string(lines[i][start:end]))
		fmt.Fprintf(&result, "%v\n", line)
	}
	paragraphs := strings.Split(result.String(), "\n\n")
	result.Reset()
	for i := range paragraphs {
		paragraphs[i] = strings.Join(strings.Fields(paragraphs[i]), " ")
	}
	return strings.Trim(strings.Join(paragraphs, "\n\n"), "\n")
}
