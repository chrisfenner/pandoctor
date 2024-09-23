package pipetable

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
			if err := w.NextRow(); err != nil {
				t.Fatalf("NextRow() = %v", err)
			}
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
