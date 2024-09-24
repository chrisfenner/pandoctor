// Package main implements the business logic for pandoctor.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	file = flag.String("file", "", "file to update in place")
)

func main() {
	flag.Parse()
	if err := mainErr(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func mainErr() error {
	args := flag.Args()
	if len(args) < 1 {
		return errors.New("please provide an action")
	}

	f, err := os.OpenFile(*file, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var newContents []byte
	action := strings.ToLower(args[0])
	switch action {
	case "convert_tables":
		newContents, err = convertTables(contents)
	default:
		return fmt.Errorf("unknown action: %q", action)
	}
	if err != nil {
		return err
	}

	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	if _, err := f.Write(newContents); err != nil {
		return err
	}
	return nil
}
