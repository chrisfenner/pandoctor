package gridtable

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadBasicTable(t *testing.T) {
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
