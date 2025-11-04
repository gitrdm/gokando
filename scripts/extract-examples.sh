#!/usr/bin/env bash
set -euo pipefail

# Extract Go Example functions and their // Output: comments into
# per-example Markdown snippets under docs/examples-snippets/.

outdir="docs/examples-snippets"
mkdir -p "$outdir"

# Find files that contain Example functions
grep -R --line-number "^func Example" --include="*.go" . | cut -d: -f1 | sort -u | while read -r f; do
  # Use awk to extract each Example function, preserve source and output block.
  awk -v fname="$f" -v outdir="$outdir" '
  function write_md(orig_fname, exname, buf,    fname_s, n, lines, i, start, md, outpath) {
    # sanitize filename
    fname_s = orig_fname
    sub(/^\.\//, "", fname_s)
    gsub(/\//, "_", fname_s)

    # build markdown
    md = "```go\n" buf "\n```\n\n"

    # split to lines and find Output block if present
    n = split(buf, lines, "\n")
    start = 0
    for (i = 1; i <= n; i++) {
      if (lines[i] ~ /^\/\/ Output:/) { start = i; break }
    }
    if (start) {
      md = md "Output:\n\n```\n"
      for (i = start; i <= n; i++) {
        if (lines[i] ~ /^\/\//) {
          s = lines[i]
          sub(/^\/\/ ?/, "", s)
          md = md s "\n"
        } else {
          break
        }
      }
      md = md "```\n"
    }

    outpath = outdir "/" fname_s "-" exname ".md"
    # ensure output directory exists
    cmd = "mkdir -p \"" outdir "\""
    system(cmd)
    print md > outpath
    close(outpath)
    print "wrote " outpath > "/dev/stderr"
  }

  BEGIN { infunc=0; brace=0; buf=""; exname="" }
  {
    if (infunc==0 && $0 ~ /^func[[:space:]]+Example/) {
      infunc=1
      buf = $0 "\n"
      if (match($0, /func[[:space:]]+([A-Za-z0-9_]+)[[:space:]]*\(/, m)) exname = m[1]; else exname = "Example"
      brace += gsub(/\{/, "{")
      brace -= gsub(/\}/, "}")
      next
    }
    if (infunc==1) {
      buf = buf $0 "\n"
      brace += gsub(/\{/, "{")
      brace -= gsub(/\}/, "}")
      if (brace==0) {
        write_md(fname, exname, buf)
        infunc=0; buf=""; exname=""
      }
    }
  }
' "$f"
done

echo "Extraction complete. Snippets in $outdir"
