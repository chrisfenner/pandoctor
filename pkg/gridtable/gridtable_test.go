package gridtable

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Small helper for finding an element in a slice.
func contains[T comparable](ts []T, elem T) bool {
	for _, t := range ts {
		if t == elem {
			return true
		}
	}
	return false
}

// Build a very simple 2 by 2 table of 1 character each
func TestBasicTable(t *testing.T) {
	for i, tc := range []struct {
		config Config
		want   string
	}{
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			want: `+---+---+
| A | B |
+---+---+
| C | D |
+---+---+
`,
		},
		{
			config: Config{
				NumHeaderRows: 1,
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			want: `+---+---+
| A | B |
+===+===+
| C | D |
+---+---+
`,
		},
		{
			config: Config{
				NumHeaderRows: 2,
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			want: `+---+---+
| A | B |
+---+---+
| C | D |
+===+===+
`,
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			w, err := NewWriter(tc.config)
			if err != nil {
				t.Fatalf("NewWriter() = %v", err)
			}
			if err := w.WriteColumn(0, Cell{
				Text: "A",
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			if err := w.WriteColumn(1, Cell{
				Text: "B",
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			w.NextRow()
			if err := w.WriteColumn(0, Cell{
				Text: "C",
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			if err := w.WriteColumn(1, Cell{
				Text: "D",
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}

			got, err := w.String()
			if err != nil {
				t.Fatalf("String() = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("String() =\n%v\nwant:\n%v\ndiff (-want +got)\n%v", got, tc.want, diff)
			}
		})
	}
}

// Build some 2 by 2 tables with row spans.
func TestSmallRowSpanTable(t *testing.T) {
	for i, tc := range []struct {
		config        Config
		rowsWithSpans []int
		want          string
	}{
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{0},
			want: `+---+---+
| A     |
+---+---+
| C | D |
+---+---+
`,
		},
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{1},
			want: `+---+---+
| A | B |
+---+---+
| C     |
+---+---+
`,
		},
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{0, 1},
			want: `+---+---+
| A     |
+---+---+
| C     |
+---+---+
`,
		},
		{
			config: Config{
				NumHeaderRows: 1,
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{0},
			want: `+---+---+
| A     |
+===+===+
| C | D |
+---+---+
`,
		},
		{
			config: Config{
				NumHeaderRows: 2,
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{0},
			want: `+---+---+
| A     |
+---+---+
| C | D |
+===+===+
`,
		},
		{
			config: Config{
				NumHeaderRows: 2,
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			rowsWithSpans: []int{1},
			want: `+---+---+
| A | B |
+---+---+
| C     |
+===+===+
`,
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			w, err := NewWriter(tc.config)
			if err != nil {
				t.Fatalf("NewWriter() = %v", err)
			}
			cell := Cell{
				Text: "A",
			}
			if contains(tc.rowsWithSpans, 0) {
				cell.ColSpan = 1
			}
			if err := w.WriteColumn(0, cell); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			if !contains(tc.rowsWithSpans, 0) {
				if err := w.WriteColumn(1, Cell{
					Text: "B",
				}); err != nil {
					t.Fatalf("WriteColumn() = %v", err)
				}
			}
			w.NextRow()
			cell = Cell{
				Text: "C",
			}
			if contains(tc.rowsWithSpans, 1) {
				cell.ColSpan = 1
			}
			if err := w.WriteColumn(0, cell); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			if !contains(tc.rowsWithSpans, 1) {
				if err := w.WriteColumn(1, Cell{
					Text: "D",
				}); err != nil {
					t.Fatalf("WriteColumn() = %v", err)
				}
			}

			got, err := w.String()
			if err != nil {
				t.Fatalf("String() = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("String() =\n%v\nwant:\n%v\ndiff (-want +got)\n%v", got, tc.want, diff)
			}
		})
	}
}

// Build some 2 by 2 tables with col spans.
func TestSmallColSpanTable(t *testing.T) {
	for i, tc := range []struct {
		config        Config
		colsWithSpans []int
		want          string
	}{
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			colsWithSpans: []int{0},
			want: `+---+---+
| A | B |
+   +---+
|   | D |
+---+---+
`,
		},
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			colsWithSpans: []int{1},
			want: `+---+---+
| A | B |
+---+   +
| C |   |
+---+---+
`,
		},
		{
			config: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
			colsWithSpans: []int{0, 1},
			want: `+---+---+
| A | B |
+   +   +
|   |   |
+---+---+
`,
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			w, err := NewWriter(tc.config)
			if err != nil {
				t.Fatalf("NewWriter() = %v", err)
			}
			cell := Cell{
				Text: "A",
			}
			if contains(tc.colsWithSpans, 0) {
				cell.RowSpan = 1
			}
			if err := w.WriteColumn(0, cell); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			cell = Cell{
				Text: "B",
			}
			if contains(tc.colsWithSpans, 1) {
				cell.RowSpan = 1
			}
			if err := w.WriteColumn(1, cell); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			w.NextRow()
			if !contains(tc.colsWithSpans, 0) {
				if err := w.WriteColumn(0, Cell{
					Text: "C",
				}); err != nil {
					t.Fatalf("WriteColumn() = %v", err)
				}
			}
			if !contains(tc.colsWithSpans, 1) {
				if err := w.WriteColumn(1, Cell{
					Text: "D",
				}); err != nil {
					t.Fatalf("WriteColumn() = %v", err)
				}
			}

			got, err := w.String()
			if err != nil {
				t.Fatalf("String() = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("String() =\n%v\nwant:\n%v\ndiff (-want +got)\n%v", got, tc.want, diff)
			}
		})
	}
}

func TestWordWrap(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 10},
			{Width: 10},
		},
	}
	for i, tc := range []struct {
		a    string
		b    string
		want string
	}{
		{
			a: "lorem",
			b: "ipsum",
			want: `+----------+----------+
| lorem    | ipsum    |
+----------+----------+
`,
		},
		{
			a: "lorem ipsum",
			b: "dolor",
			want: `+----------+----------+
| lorem    | dolor    |
| ipsum    |          |
+----------+----------+
`,
		},
		{
			a: "lorem ipsum",
			b: "dolor sit amet",
			want: `+----------+----------+
| lorem    | dolor    |
| ipsum    | sit amet |
+----------+----------+
`,
		},
		{
			a: "lorem",
			b: "ipsum dolor sit amet",
			want: `+----------+----------+
| lorem    | ipsum    |
|          | dolor    |
|          | sit amet |
+----------+----------+
`,
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			w, err := NewWriter(config)
			if err != nil {
				t.Fatalf("NewWriter() = %v", err)
			}

			if err := w.WriteColumn(0, Cell{
				Text: tc.a,
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			if err := w.WriteColumn(1, Cell{
				Text: tc.b,
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}

			got, err := w.String()
			if err != nil {
				t.Fatalf("String() = %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("String() =\n%v\nwant:\n%v\ndiff (-want +got)\n%v", got, tc.want, diff)
			}
		})
	}
}

func TestFailedWordWrap(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 10},
		},
	}
	for i, tc := range []struct {
		a string
	}{
		{
			a: "loremipsumdolorsitamet",
		},
		{
			a: "loremipsum",
		},
		{
			a: "loremipsu",
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			w, err := NewWriter(config)
			if err != nil {
				t.Fatalf("NewWriter() = %v", err)
			}

			if err := w.WriteColumn(0, Cell{
				Text: tc.a,
			}); err != nil {
				t.Fatalf("WriteColumn() = %v", err)
			}
			want := ErrBadWrap
			if _, err := w.String(); !errors.Is(err, want) {
				t.Errorf("String() = %v, want %v", err, want)
			}
		})
	}
}

func TestColumnIndexOutOfRange(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	want := ErrColumnIndexOutOfRange
	if err := w.WriteColumn(2, Cell{
		Text: "C",
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
	if err := w.WriteColumn(1, Cell{
		Text:    "B",
		ColSpan: 1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
	if err := w.WriteColumn(0, Cell{
		Text:    "A",
		ColSpan: 2,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestWroteShadowedCell(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	if err := w.WriteColumn(0, Cell{
		Text:    "A",
		ColSpan: 1,
	}); err != nil {
		t.Fatalf("WriteColumn() = %v", err)
	}
	want := ErrShadowedCell
	if err := w.WriteColumn(1, Cell{
		Text: "B",
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestShadowedWrittenCell(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	if err := w.WriteColumn(1, Cell{
		Text: "B",
	}); err != nil {
		t.Errorf("WriteColumn() = %v", err)
	}
	want := ErrShadowedCell
	if err := w.WriteColumn(0, Cell{
		Text:    "A",
		ColSpan: 1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestSpanOutOfHeader(t *testing.T) {
	config := Config{
		NumHeaderRows: 1,
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	want := ErrSpanBeyondHeader
	if err := w.WriteColumn(0, Cell{
		Text:    "A",
		RowSpan: 1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestOverlappingSpans(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	if err := w.WriteColumn(1, Cell{
		Text:    "B",
		RowSpan: 1,
	}); err != nil {
		t.Fatalf("WriteColumn() = %v", err)
	}
	w.NextRow()
	want := ErrOverlappingSpans
	if err := w.WriteColumn(0, Cell{
		Text:    "C",
		ColSpan: 1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestNegativeSpans(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{
			{Width: 3},
			{Width: 3},
		},
	}
	w, err := NewWriter(config)
	if err != nil {
		t.Fatalf("NewWriter() = %v", err)
	}
	want := ErrNegativeSpan
	if err := w.WriteColumn(1, Cell{
		Text:    "B",
		RowSpan: -1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
	if err := w.WriteColumn(1, Cell{
		Text:    "B",
		ColSpan: -1,
	}); !errors.Is(err, want) {
		t.Errorf("WriteColumn() = %v, want %v", err, want)
	}
}

func TestInvalidColumnSpec(t *testing.T) {
	config := Config{
		Columns: []ColumnSpec{},
	}
	want := ErrInvalidColumnSpec
	if _, err := NewWriter(config); !errors.Is(err, want) {
		t.Errorf("NewWriter() = %v, want %v", err, want)
	}
}
