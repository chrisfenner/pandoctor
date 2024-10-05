package gridtable

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadTable(t *testing.T) {
	for i, tc := range []struct {
		str        string
		want       [][]*Cell
		wantConfig Config
	}{
		{
			str: `+---+---+
| A | B |
+---+---+
| C | D |
+---+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
		{
			str: `+---+---+
| A | B |
+===+===+
| C | D |
+---+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
				NumHeaderRows: 1,
			},
		},
		{
			str: `+---+----+
| A | B  |
+---+----+
| C |  D |
+---+----+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 4},
				},
			},
		},
		{
			str: `+----+---+
|  A | B |
+----+---+
| C  | D |
+----+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 4},
					{Width: 3},
				},
			},
		},
		{
			str: `+---+---+
| A |   |
|   | B |
+---+---+
| C | D |
+---+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
		{
			str: `+---+---+
| A | B |
+---+---+
|   | D |
| C |   |
+---+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
		// More complex tables, with spans.
		{
			str: `+---+---+
| A     |
+---+---+
| C | D |
+---+---+
`,
			want: [][]*Cell{
				{
					{
						Text:    "A",
						ColSpan: 1,
					},
					nil,
				},
				{
					{Text: "C"},
					{Text: "D"},
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
		{
			str: `+---+---+
| A     |
+---+---+
| C     |
+---+---+
`,
			want: [][]*Cell{
				{
					{
						Text:    "A",
						ColSpan: 1,
					},
					nil,
				},
				{
					{
						Text:    "C",
						ColSpan: 1,
					},
					nil,
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
		{
			str: `+---+---+
| A | B |
+---+---+
| C     |
+---+---+
`,
			want: [][]*Cell{
				{
					{Text: "A"},
					{Text: "B"},
				},
				{
					{
						Text:    "C",
						ColSpan: 1,
					},
					nil,
				},
			},
			wantConfig: Config{
				Columns: []ColumnSpec{
					{Width: 3},
					{Width: 3},
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			r, err := NewReader(bytes.NewReader([]byte(tc.str)))
			if err != nil {
				t.Fatalf("NewReader() = %v", err)
			}
			var got [][]*Cell
			for cells, err := range r.Read() {
				if err != nil {
					t.Fatalf("Read() = %v", err)
				}
				got = append(got, cells)
			}
			if !cmp.Equal(got, tc.want) {
				t.Errorf("got %v\nwant %v", got, tc.want)
			}
			gotConfig, err := r.GetConfig()
			if err != nil {
				t.Fatalf("GetConfig() = %v", err)
			}
			if !cmp.Equal(*gotConfig, tc.wantConfig) {
				t.Errorf("GetConfig() = %v\nwant %v", gotConfig, tc.wantConfig)
			}
		})
	}
}

func TestReadMalformedTable(t *testing.T) {
	for i, tc := range []string{
		`+---+!--+
| A | B |
+---+---+
| C | D |
+---+---+
`,
		`+---+---+
| A | B
+---+---+
| C | D |
+---+---+
`,
		`+---+---+
| A | B |
+---+-+-+
| C | D |
+---+---+
`,
		`+---+---+
| A | B |
+---+---+
| C | D |
+---+----+
`,
		`+---+---+
| A | B |
+   +---+
|   | D |
+---+----+
`,
		`+===+===+
| A | B |
+---+---+
| C | D |
+---+---+
`,
		`+---+---+
| A | B |
+===+===+
| C | D |
+===+===+
`,
	} {
		t.Run(fmt.Sprintf("table_%v", i), func(t *testing.T) {
			r, err := NewReader(bytes.NewReader([]byte(tc)))
			if err == nil {
				for _, err = range r.Read() {
					if err != nil {
						break
					}
				}
			}
			if !errors.Is(err, ErrMalformedTable) {
				t.Errorf("got %v want %v", err, ErrMalformedTable)
			}
		})
	}
}
