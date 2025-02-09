package main

import (
	"fmt"
	"io"
	"text/tabwriter"
)

func usage(out io.Writer, full bool) {
	if full {
		_, _ = fmt.Fprintf(out, "\n  "+bold.Render("walk "+Version)+"\n\n  Usage: walk [path]\n\n")
	}
	w := tabwriter.NewWriter(out, 0, 8, 2, ' ', 0)
	put := func(s string) {
		_, _ = fmt.Fprintln(w, s)
	}
	put("    arrows, hjkl\tMove cursor")
	put("    enter\tEnter directory")
	put("    backspace\tExit directory")
	put("    space\tToggle preview")
	put("    esc, q\tExit with cd")
	put("    ctrl+c\tExit without cd")
	put("    /\tFuzzy search")
	put("    d, delete\tDelete file or dir")
	put("    y\tCopy to clipboard")
	put("    .\tHide hidden files")
	put("    ?\tShow help")
	if full {
		put("\n  Flags:\n")
		put("    --icons\tdisplay icons")
		put("    --dir-only\tshow dirs only")
		put("    --hide-hidden\thide hidden files")
		put("    --preview\tdisplay preview")
		put("    --with-border\tpreview with border")
		put("    --fuzzy\tfuzzy mode")
	}
	_ = w.Flush()
	_, _ = fmt.Fprintf(out, "\n")
}
