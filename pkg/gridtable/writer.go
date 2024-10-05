package gridtable

import (
	"fmt"
	"strings"
)

// A Config configures the initialization of a Writer.
type Config struct {
	// The top `NumHeaderRows` are considered to be the header of the table.
	// 0 = no header.
	NumHeaderRows int
	// The specification of the columns in the table.
	Columns []ColumnSpec
}

// Writer is an object that can be used to write out a grid table.
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
	w.NextRow()
	return w, nil
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

	// Check that the cell row span doesn't straddle the header boundary (if any).
	if w.config.NumHeaderRows != 0 {
		if w.currentRow < w.config.NumHeaderRows && w.currentRow+cell.RowSpan >= w.config.NumHeaderRows {
			return fmt.Errorf(
				"%w: cell at row %d, column %d spanned %d rows, but the header is only %d rows",
				ErrSpanBeyondHeader, w.currentRow, index, cell.RowSpan+1, w.config.NumHeaderRows)
		}
	}

	// Check for shadowing errors.
	if w.shadowed[w.currentRow][index] {
		return fmt.Errorf("%w: wrote to shadowed cell at row %d, column %d", ErrShadowedCell, w.currentRow, index)
	}
	for j := index + 1; j <= index+cell.ColSpan; j++ {
		if w.written[w.currentRow][j] {
			return fmt.Errorf(
				"%w: cell at row %d, column %d with span %d shadowed previously-written cell at row %d, column %d",
				ErrShadowedCell, w.currentRow, index, cell.ColSpan, w.currentRow, j)
		}
	}
	for i := w.currentRow; i <= w.currentRow+cell.RowSpan; i++ {
		if i >= len(w.shadowed) {
			break
		}
		for j := index; j <= index+cell.ColSpan; j++ {
			if w.shadowed[i][j] {
				return fmt.Errorf("%w: two spans overlapped at row %d, column %d", ErrOverlappingSpans, i, j)
			}
		}
	}

	// Everything is OK. Write the cell.
	w.cells[w.currentRow][index] = cell
	w.written[w.currentRow][index] = true
	// Record the newly shadowed cells, extending the array if needed.
	for i := w.currentRow; i < w.currentRow+cell.RowSpan+1; i++ {
		if i >= len(w.shadowed) {
			w.shadowed = append(w.shadowed, make([]bool, len(w.config.Columns)))
		}
		for j := index; j < index+cell.ColSpan+1; j++ {
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
func (w *Writer) NextRow() {
	w.currentRow++
	// The `shadowed` array might have been extended past this point already due to spans.
	// Only extend it here if this is not the case.
	if len(w.shadowed) == len(w.cells) {
		w.shadowed = append(w.shadowed, make([]bool, len(w.config.Columns)))
	}
	w.cells = append(w.cells, make([]Cell, len(w.config.Columns)))
	w.written = append(w.written, make([]bool, len(w.config.Columns)))
}

// String writes out the table to a string.
func (w *Writer) String() (string, error) {
	// Convenience:
	// If the caller called NextRow() and then String(), don't show them an empty row.
	lastRow := len(w.cells) - 1
	anyWritten := false
	for j := range w.config.Columns {
		if w.written[lastRow][j] || w.shadowed[lastRow][j] {
			anyWritten = true
		}
	}
	if !anyWritten {
		w.cells = w.cells[:len(w.cells)-1]
	}

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
				for _, rowHeight := range rowHeights[i+1 : i+w.cells[i][j].RowSpan+1] {
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

	// Draw all the boxes.
	y = 1
	for i := range w.cells {
		x = 1
		for j := range w.config.Columns {
			// Draw the +'s in the box around this cell.
			// Note that array[x][y] is the top left inside of the cell.
			array[x-1][y-1] = '+'
			array[x+w.config.Columns[j].Width][y-1] = '+'
			array[x-1][y+rowHeights[i]] = '+'
			array[x+w.config.Columns[j].Width][y+rowHeights[i]] = '+'

			// Draw the |'s to the right of this cell and the -'s (='s if header) below this cell.
			for n := y; n < y+rowHeights[i]; n++ {
				array[x+w.config.Columns[j].Width][n] = '|'
			}
			sep := '-'
			if w.config.NumHeaderRows != 0 && w.config.NumHeaderRows == i+1+w.cells[i][j].RowSpan {
				sep = '='
			}
			for n := x; n < x+w.config.Columns[j].Width; n++ {
				array[n][y+rowHeights[i]] = sep
			}

			x += w.config.Columns[j].Width + 1 // move the cursor to the x position of the next cell
		}
		y += rowHeights[i] + 1 // move the cursor to the y position of the next cell
	}

	// Draw the contents of all the (non-shadowed) cells.
	y = 1
	for i := range w.cells {
		x = 1
		for j := range w.config.Columns {
			if !w.shadowed[i][j] {
				if err := drawCellContents(array, x, y, i, j, &w.cells[i][j], w.config.Columns, rowHeights); err != nil {
					// We expect to have hit any errors to do with the cell contents already.
					// An error here indicates a flaw in this library itself.
					panic(fmt.Sprintf("unexpected error painting cell at row %d, column %d: %v", i, j, err))
				}
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
