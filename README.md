# pandoctor
Tools for fixing up Pandoc Markdown files

Note: This tool is extremely work-in-progress!

## Usage

### Converting HTML tables to Markdown

`convert_tables` will parse the HTML tables in the file and replace them with
grid tables.

```sh
pandoctor --file /path/to/your/markdown/file convert_tables
```

You can use the optional `--table_width` flag (default 120 chars) to control
the total width of the table.

```sh
pandoctor --file /path/to/your/markdown/file --table_width 100 convert_tables
```

By default, Pandoctor will replace tables it couldn't convert with a message
explaining what went wrong. You can use `--ignore_errors` to suppress this and
just leave those tables alone.

### Resizing Markdown grid tables

`resize_tables` will take existing grid tables and resize them to
`--new_widths` if they match a given column description (`--match_columns`).

```sh
pandoctor --file= /path/to/your/markdown/file  --match_columns headinga,headingb,headingc --new_widths 10,20,30 resize_tables
```

By default, Pandoctor will replace tables it couldn't resize with a message
explaining what went wrong. You can use `--ignore_errors` to suppress this and
just leave those tables alone.
