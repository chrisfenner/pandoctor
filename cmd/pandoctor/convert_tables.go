package main

import (
	"bytes"
	"flag"
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"

	"github.com/chrisfenner/pandoctor/pkg/gridtable"
	"golang.org/x/net/html"
)

var (
	tableWidth = flag.Int("table_width", 120, "width of output tables")
	ignoreErrors = flag.Bool("ignore_errors", false, "set to leave a table as-is if there is an error")
)

func convertTables(contents []byte) ([]byte, error) {
	tableRe := regexp.MustCompile("<table.*>\n(<.*\n)*</table>")
	return tableRe.ReplaceAllFunc(contents, rewriteHTMLTableAsGrid), nil
}

func getTableNode(contents []byte) (*html.Node, error) {
	parent, err := html.Parse(bytes.NewReader(contents))
	if err != nil {
		return nil, err
	}
	// first child should be an html element
	doc := parent.FirstChild
	if doc == nil {
		return nil, fmt.Errorf("html.Parse didn't return an <html> element")
	}
	// html should have a body
	var body *html.Node
	for child := range children(doc) {
		if child.Type == html.ElementNode && child.Data == "body" {
			body = child
		}
	}
	if body == nil {
		return nil, fmt.Errorf("html.Parse didn't return a <body> element")
	}
	// body should have a table
	for child := range children(body) {
		if child.Type == html.ElementNode && child.Data == "table" {
			return child, nil
		}
	}
	return nil, fmt.Errorf("html.Parse didn't return a <table> element")

}

func rewriteHTMLTableAsGrid(contents []byte) []byte {
	table, err := getTableNode(contents)
	caption := ""
	id := ""
	for _, attr := range table.Attr {
		if attr.Key == "id" {
			id = attr.Val
		}
	}
	for child := range children(table) {
		if child.Type == html.ElementNode && child.Data == "caption" {
			caption = flatten(child)
		}
	}
	config, err := generateTableConfig(table)
	if err != nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not generate table config: %v", err))
	}
	w, err := gridtable.NewWriter(*config)
	if err != nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not initialize table writer: %v", err))
	}
	// find the (first) thead and (first) tbody
	var thead *html.Node
	var tbody *html.Node
	for child := range children(table) {
		if child.Type == html.ElementNode {
			if child.Data == "thead" {
				thead = child
			}
			if child.Data == "tbody" {
				tbody = child
			}
		}
	}
	if tbody == nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not parse table: no <tbody> was found"))
	}
	for tr := range children(thead) {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}
		i := 0
		for td := range children(tr) {
			if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
				continue
			}
			colspan, err := numericAttribute(td.Attr, "colspan")
			if err != nil {
				if *ignoreErrors {
					return contents
				}
				return []byte(fmt.Sprintf("Could not parse colspan: %v", err))
			}
			rowspan, err := numericAttribute(td.Attr, "rowspan")
			if err != nil {
				if *ignoreErrors {
					return contents
				}
				return []byte(fmt.Sprintf("Could not parse rowspan: %v", err))
			}
			cell := gridtable.Cell{
				Text:    flatten(td),
				RowSpan: rowspan,
				ColSpan: colspan,
			}
			if err := w.WriteColumn(i, cell); err != nil {
				if *ignoreErrors {
					return contents
				}
				return []byte(fmt.Sprintf("Could not write cell: %v", err))
			}
			i += colspan
			i++
		}
		w.NextRow()
	}
	for tr := range children(tbody) {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}
		i := 0
		for td := range children(tr) {
			if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
				continue
			}
			colspan, err := numericAttribute(td.Attr, "colspan")
			if err != nil {
				if *ignoreErrors {
					return contents
				}
				return []byte(fmt.Sprintf("Could not parse colspan: %v", err))
			}
			rowspan, err := numericAttribute(td.Attr, "rowspan")
			if err != nil {
				if *ignoreErrors {
					return contents
				}
				return []byte(fmt.Sprintf("Could not parse rowspan: %v", err))
			}
			if rowspan != 0 {
				if *ignoreErrors {
					return contents
				}
				return []byte("rowspan not currently supported")
			}
			if colspan != 0 {
				// HTML uses colspan=3 to represent a row that spans 3 columns total.
				colspan -= 1
			}
			cell := gridtable.Cell{
				Text:    flatten(td),
				ColSpan: colspan,
			}
			cell.Text = flatten(td)
			w.WriteColumn(i, cell)
			i += colspan
			i++
		}
		w.NextRow()
	}
	var sb strings.Builder
	sb.WriteString("Table:")
	if caption != "" {
		fmt.Fprintf(&sb, " %v", caption)
	}
	if id != "" {
		fmt.Fprintf(&sb, " {#%v}", id)
	}
	sb.WriteString("\n\n")
	result, err := w.String()
	if err != nil {
		if *ignoreErrors {
			return contents
		}
		return []byte(fmt.Sprintf("Could not render grid table: %v", err))
	}
	sb.WriteString(result)
	return []byte(sb.String())
}

func flatten(node *html.Node) string {
	if node == nil {
		return ""
	}
	var sb strings.Builder
	if node.Type == html.ElementNode {
		// Special cases (open)
		if node.Data == "em" {
			sb.WriteString("*")
		} else if node.Data == "strong" {
			sb.WriteString("**")
		}
		for child := range children(node) {
			sb.WriteString(flatten(child))
		}
		// Special cases (close)
		if node.Data == "em" {
			sb.WriteString("*")
		} else if node.Data == "strong" {
			sb.WriteString("**")
		}

		// add an empty line between paragraphs.
		if node.Data == "p" && node.NextSibling != nil {
			sb.WriteString("\n")
		}
	} else if node.Type == html.TextNode {
		sb.WriteString(node.Data)
	}
	return sb.String()
}

func widthForColumn(attrs []html.Attribute) (int, error) {
	for _, attr := range attrs {
		if attr.Key == "style" {
			for _, styleAttr := range strings.Split(attr.Val, ",") {
				kv := strings.Split(styleAttr, ":")
				if len(kv) != 2 {
					continue
				}
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				if k == "width" {
					val := strings.ReplaceAll(v, "%", "")
					pct, err := strconv.ParseInt(val, 10, 32)
					if err != nil {
						return 0, err
					}
					return int(pct), nil
				}
			}
		}
	}
	// Width was not found, use a reasonable default.
	return 1, nil
}

func numericAttribute(attrs []html.Attribute, key string) (int, error) {
	for _, attr := range attrs {
		if attr.Key == key {
			val, err := strconv.ParseInt(attr.Val, 10, 32)
			if err != nil {
				return 0, err
			}
			return int(val), nil
		}
	}
	// Attribute was not found.
	return 0, nil
}

// children is an iterator that iterates the children of the given Node.
func children(node *html.Node) iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		if node == nil {
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if !yield(child) {
				return
			}
		}
		return
	}
}

func generateTableConfig(table *html.Node) (*gridtable.Config, error) {
	numHeaderRows := 0
	var colWidths []int
	for child := range children(table) {
		if child.Type == html.ElementNode {
			// Iterate the <colgroup> child <col> elements to find the column widths.
			if child.Data == "colgroup" {
				for col := range children(child) {
					if col.Type == html.ElementNode && col.Data == "col" {
						width, err := widthForColumn(col.Attr)
						if err != nil {
							return nil, err
						}
						colWidths = append(colWidths, width)
					}
				}
			}
			// Iterate the <thead> child <tr> elements to find the number of headers.
			if child.Data == "thead" {
				for tr := range children(child) {
					if tr.Type == html.ElementNode && tr.Data == "tr" {
						numHeaderRows++
					}
				}
			}
		}
	}

	// normalize the widths against the table's width (minus separator symbols)
	totalTableWidth := *tableWidth - len(colWidths) - 1
	result := gridtable.Config{
		NumHeaderRows: numHeaderRows,
		Columns:       make([]gridtable.ColumnSpec, 0, len(colWidths)),
	}
	totalColWidth := 0
	for _, colWidth := range colWidths {
		totalColWidth += colWidth
	}
	remainingTableWidth := totalTableWidth
	if len(colWidths) > 1 {
		for _, colWidth := range colWidths[1:] {
			thisColumnWidth := colWidth * totalTableWidth / totalColWidth
			result.Columns = append(result.Columns, gridtable.ColumnSpec{
				Width: thisColumnWidth,
			})
			remainingTableWidth -= thisColumnWidth
		}
	}
	// Use up all the remaining space on the first column
	result.Columns = append([]gridtable.ColumnSpec{
		{
			Width: remainingTableWidth,
		},
	}, result.Columns...)

	return &result, nil
}
