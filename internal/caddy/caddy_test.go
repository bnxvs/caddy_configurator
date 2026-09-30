package caddy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCaddy writes a fake `caddy` executable understanding
// `fmt --overwrite FILE` and `validate --config FILE --adapter caddyfile`,
// and prepends its dir to PATH.
func fakeCaddy(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
if [ "$1" = "fmt" ]; then
  file="$3"
  content=$(cat "$file")
  printf 'formatted-by-fake\n%s' "$content" > "$file"
  exit 0
fi
if [ "$1" = "validate" ]; then
  file="$3"
  if grep -q "INVALID" "$file"; then
    echo "fake: invalid Caddyfile" >&2
    exit 1
  fi
  echo "fake: valid configuration"
  exit 0
fi
exit 2
`
	p := filepath.Join(dir, "caddy")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func noCaddy(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir()) // empty dir: no caddy resolvable
}

func TestFormatWithBinary(t *testing.T) {
	fakeCaddy(t)
	if !Found() {
		t.Fatal("Found() = false with fake caddy on PATH")
	}
	out, applied := FormatCaddyfile("a.example.com {\n\trespond ok\n}\n")
	if !applied {
		t.Fatal("applied = false, want true")
	}
	if !strings.HasPrefix(out, "formatted-by-fake\n") {
		t.Fatalf("out = %q", out)
	}
}

func TestFormatWithoutBinary(t *testing.T) {
	noCaddy(t)
	if Found() {
		t.Fatal("Found() = true with empty PATH dir")
	}
	const in = "a.example.com {\n\trespond ok\n}\n"
	out, applied := FormatCaddyfile(in)
	if applied || out != in {
		t.Fatalf("out = %q applied = %v, want input untouched", out, applied)
	}
}

func TestValidateWithBinary(t *testing.T) {
	fakeCaddy(t)
	ok := ValidateCaddyfile("a.example.com {\n\trespond ok\n}\n")
	if !ok.Available || !ok.OK {
		t.Fatalf("result = %+v, want available+ok", ok)
	}
	bad := ValidateCaddyfile("INVALID {{{")
	if !bad.Available || bad.OK {
		t.Fatalf("result = %+v, want available+not-ok", bad)
	}
	if !strings.Contains(bad.Output, "fake: invalid") {
		t.Fatalf("output = %q", bad.Output)
	}
}

func TestValidateWithoutBinary(t *testing.T) {
	noCaddy(t)
	res := ValidateCaddyfile("whatever")
	if res.Available {
		t.Fatalf("result = %+v, want unavailable", res)
	}
}
