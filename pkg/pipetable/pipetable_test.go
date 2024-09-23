package pipetable

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Build a very simple 2 by 2 table of 1 character each
func TestBasicTable(t *testing.T) {
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
	want := `+---+---+
| A | B |
+---+---+
| C | D |
+---+---+
`
	got, err := w.String()
	if err != nil {
		t.Fatalf("String() = %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("String() =\n%v\nwant:\n%v\ndiff (-want +got)\n%v", got, want, diff)
	}
}
