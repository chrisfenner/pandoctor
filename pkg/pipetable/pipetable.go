// Package pipetable implements a library for printing pipe tables.
package pipetable

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
	// ErrInvalidColumnSpec indicates that a column spec was invalid.
	ErrInvalidColumnSpec = errors.New("invalid column spec")
	// ErrBadWrap indicates that text could not be wrapped to fit into its column.
	ErrBadWrap = errors.New("text could not be wrapped")
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

// A Config configures the initialization of a PipeTableWriter
type Config struct {
	Columns []ColumnSpec
}

// Writer is an object that can be used to write out a pipe table.
type Writer struct {
	config     Config
	currentRow int
	// Cell data for the current row.
	// cells[i][j] is the j'th column of the i'th row.
	cells [][]Cell
	// Cells which have been written.
	// written[i][j] is the j'th column of the i'th row.
	written [][]bool
	// Cells which are "shadowed" by spanned cells written previously.
	// shadowed[i][j] is the j'th column of the i'th row.
	shadowed [][]bool
}

// NewWriter initializes a new Writer based on the specified configuration.
// Rows are written to `out` as they become ready.
func NewWriter(config Config) (*Writer, error) {
	if len(config.Columns) < 1 {
		return nil, fmt.Errorf("%w: table needs at least 1 column", ErrInvalidColumnSpec)
	}
	for j, columnSpec := range config.Columns {
		if columnSpec.Width < minColumnWidth {
			return nil, fmt.Errorf("%w: column %d has width %d (minimum: %d)", ErrInvalidColumnSpec, j, columnSpec.Width, minColumnWidth)
		}
	}
	w := &Writer{
		config:     config,
		currentRow: -1,
	}
	if err := w.NextRow(); err != nil {
		return nil, err
	}
	return w, nil
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

// WriteColumn writes the cell into the specified column of the current row.
func (w *Writer) WriteColumn(index int, cell Cell) error {
	// Basic column indexing.
	if index < 0 {
		return fmt.Errorf("%w: %d", ErrColumnIndexOutOfRange, index)
	}
	if index >= len(w.config.Columns) {
		return fmt.Errorf("%w: %d (max is %d)", ErrColumnIndexOutOfRange, index, len(w.config.Columns))
	}

	// Check that the cell span isn't negative.
	if cell.ColSpan < 0 {
		return fmt.Errorf(
			"%w: cell at row %d, column %d had a negative ColSpan",
			ErrNegativeSpan, w.currentRow, index)
	}
	if cell.RowSpan < 0 {
		return fmt.Errorf(
			"%w: cell at row %d, column %d had a negative RowSpan",
			ErrNegativeSpan, w.currentRow, index)
	}

	// Check that cell column span doesn't go farther than the last column of the table.
	if index+cell.ColSpan >= len(w.config.Columns) {
		return fmt.Errorf(
			"%w: cell at row %d, column %d spanned %d columns, but the table has only %d",
			ErrColumnIndexOutOfRange, w.currentRow, index, cell.ColSpan+1, len(w.config.Columns))
	}

	// Check for shadowing errors.
	if w.shadowed[w.currentRow][index] {
		return fmt.Errorf("%w: wrote to shadowed cell at row %d, column %d", ErrShadowedCell, w.currentRow, index)
	}
	for j := index + 1; j < index+cell.ColSpan; j++ {
		if w.written[w.currentRow][j] {
			return fmt.Errorf(
				"%w: cell at row %d, column %d with span %d shadowed previously-written cell at row %d, column %d",
				ErrShadowedCell, w.currentRow, index, cell.ColSpan, w.currentRow, j)
		}
	}
	for i := w.currentRow; i < w.currentRow+cell.RowSpan; i++ {
		if i >= len(w.shadowed) {
			break
		}
		for j := index; j < index+cell.ColSpan; j++ {
			if w.shadowed[i][j] {
				return fmt.Errorf("%w: two spans overlapped at row %d, column %d", ErrOverlappingSpans, i, j)
			}
		}
	}

	// Everything is OK. Write the cell.
	w.cells[w.currentRow][index] = cell
	w.written[w.currentRow][index] = true
	// Record the newly shadowed cells, extending the array if needed.
	for i := w.currentRow; i < w.currentRow+cell.RowSpan; i++ {
		if i >= len(w.shadowed) {
			w.shadowed = append(w.shadowed, make([]bool, len(w.config.Columns)))
		}
		for j := index; j < index+cell.ColSpan; j++ {
			// A cell doesn't shadow itself.
			if i == w.currentRow && j == index {
				continue
			}
			w.shadowed[i][j] = true
		}
	}
	return nil
}

// NextRow finishes the current row and moves onto the next one.
func (w *Writer) NextRow() error {
	w.currentRow++
	// The `shadowed` array might have been extended past this point already due to spans.
	// Only extend it here if this is not the case.
	if len(w.shadowed) == len(w.cells) {
		w.shadowed = append(w.shadowed, make([]bool, len(w.config.Columns)))
	}
	w.cells = append(w.cells, make([]Cell, len(w.config.Columns)))
	w.written = append(w.written, make([]bool, len(w.config.Columns)))
	return nil
}

// String writes out the table to a string.
func (w *Writer) String() (string, error) {
	// Strategy: we construct a 2D array of characters and fill it in with the content of the cells,
	// draw the boundary lines, then emit the array.

	// First, we need to compute the height of each cell.
	cellHeights := make([][]int, len(w.cells))
	for i := range cellHeights {
		cellHeights[i] = make([]int, len(w.config.Columns))
		for j := range w.config.Columns {
			height, err := calculateCellHeight(j, &w.cells[i][j], w.config.Columns)
			if err != nil {
				return "", fmt.Errorf("in row %d: %w", i, err)
			}
			cellHeights[i][j] = height
		}
	}

	// Now we can compute the height of each row.
	rowHeights := make([]int, len(w.cells))
	// First pass: each row's height is the height of the tallest non-row-spanning cell in that row.
	for i := range rowHeights {
		rowHeights[i] = 1
		for j := range cellHeights[i] {
			if w.cells[i][j].RowSpan > 0 {
				continue
			}
			if cellHeights[i][j] > rowHeights[i] {
				rowHeights[i] = cellHeights[i][j]
			}
		}
	}
	// Second pass: satisfy every row span by increasing the size of the spanned rows.
	// To balance the resulting table's attractiveness with the complexity of this code, go span-by-span
	// and expand the affected rows evenly until the span is satisfied.
	for i := range w.cells {
		for j := range w.config.Columns {
			if w.cells[i][j].ColSpan > 0 {
				heightToAdd := cellHeights[i][j] - rowHeights[i]
				for _, rowHeight := range rowHeights[i+1 : i+w.cells[i][j].ColSpan] {
					heightToAdd -= rowHeight + 1 // we save a row from the separator here, too.
				}
				if heightToAdd > 0 {
					heightToAddToEachRow := (heightToAdd + w.cells[i][j].ColSpan - 1) / w.cells[i][j].ColSpan
					for row := i; row <= i+w.cells[i][j].ColSpan; row++ {
						rowHeights[row] += heightToAddToEachRow
					}
				}
			}
		}
	}

	// Now we can allocate a 2D array of runes and fill it in.
	width := calculateTableWidth(w.config.Columns)
	height := calculateTableHeight(rowHeights)
	array := make([][]rune, width)
	for y := range array {
		array[y] = make([]rune, height)
		for x := range array[y] {
			array[y][x] = ' '
		}
	}

	// Draw the top pipes.
	array[0][0] = '+'
	x := 1
	for _, col := range w.config.Columns {
		for n := 0; n < col.Width; n++ {
			array[x][0] = '-'
			x++
		}
		array[x][0] = '+'
		x++
	}

	// Draw the left pipes.
	y := 1
	for _, rowHeight := range rowHeights {
		for n := 0; n < rowHeight; n++ {
			array[0][y] = '|'
			y++
		}
		array[0][y] = '+'
		y++
	}

	// Draw every (non-shadowed) cell, and its right and bottom pipes.
	y = 1
	for i := range w.cells {
		x = 1
		for j := range w.config.Columns {
			if !w.shadowed[i][j] {
				if err := drawCellContents(array, x, y, j, &w.cells[i][j], w.config.Columns); err != nil {
					// We expect to have hit any errors to do with the cell contents already.
					// An error here indicates a flaw in this library itself.
					panic(fmt.Sprintf("unexpected error painting cell at row %d, column %d: %v", i, j, err))
				}

			}
			// Draw the +'s in the box around this cell.
			array[x-1][y-1] = '+'
			array[x+w.config.Columns[j].Width][y-1] = '+'
			array[x-1][y+rowHeights[i]] = '+'
			array[x+w.config.Columns[j].Width][y+rowHeights[i]] = '+'

			// Draw the |'s to the right of this cell and the -'s below this cell.
			dx := cellWidth(j, &w.cells[i][j], w.config.Columns)
			dy := cellHeight(i, &w.cells[i][j], rowHeights)
			for n := y; n < y+dy; n++ {
				array[x+dx][n] = '|'
			}
			for n := x; n < x+dx; n++ {
				array[n][y+dy] = '-'
			}

			x += w.config.Columns[j].Width + 1 // move the cursor to the x position of the next cell
		}
		y += rowHeights[i] + 1 // move the cursor to the y position of the next cell
	}

	// Concatenate the array to a string and return it.
	var sb strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sb.WriteRune(array[x][y])
		}
		sb.WriteRune('\n')
	}
	return sb.String(), nil
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
		result += colSpec[j].Width
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

func drawCellContents(array [][]rune, x int, y int, column int, cell *Cell, colSpec []ColumnSpec) error {
	lines, err := lines(column, cell, colSpec)
	if err != nil {
		return err
	}
	for dx, line := range lines {
		for dy, r := range line {
			array[x+1+dx][y+dy] = r
		}
	}
	return nil
}
