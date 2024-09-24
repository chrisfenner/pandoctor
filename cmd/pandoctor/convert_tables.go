package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func convertTables(contents []byte) ([]byte, error) {
	tableRe := regexp.MustCompile("<table.*>\n(<.*\n)*</table>")
	return tableRe.ReplaceAllFunc(contents, rewriteHTMLTableAsGrid), nil
}

func rewriteHTMLTableAsGrid(contents []byte) []byte {
	tokenizer := html.NewTokenizer(bytes.NewReader(contents))
	var result strings.Builder

	for {
		tok := tokenizer.Next()
		if tok == html.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				break
			} else {
				errString := fmt.Sprintf("Error! %v", tokenizer.Err())
				return []byte(errString)
			}
		}
		if tok == html.StartTagToken {
			t := tokenizer.Token()
			if t.Data == "td" {
				inner := tokenizer.Next()
				if inner == html.TextToken {
					text := (string)(tokenizer.Text())
					t := strings.TrimSpace(text)
					fmt.Fprintf(&result, "%v, ", t)
				}
			}
		}
	}
	return []byte(result.String())
}
