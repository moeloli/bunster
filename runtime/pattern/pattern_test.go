package pattern

import (
	"fmt"
	"regexp/syntax"
	"testing"
)

var translateTests = []struct {
	pat     string
	mode    mode
	want    string
	wantErr bool
}{
	{pat: ``, want: ``},
	{pat: `foo`, want: `foo`},
	{pat: `foóà中`, mode: Filenames | Braces, want: `foóà中`},
	{pat: `.`, want: `\.`},
	{pat: `foo*`, want: `(?s)foo.*`},
	{pat: `foo*`, mode: Shortest, want: `(?s)foo.*?`},
	{pat: `foo*`, mode: Shortest | Filenames, want: `foo[^/]*?`},
	{pat: `*foo`, mode: Filenames, want: `[^/]*foo`},
	{pat: `**`, want: `(?s).*.*`},
	{pat: `**`, mode: Filenames, want: `(?s).*`},
	{pat: `/**/foo`, want: `(?s)/.*.*/foo`},
	{pat: `/**/foo`, mode: Filenames, want: `(?s)/(.*/|)foo`},
	{pat: `/**/à`, mode: Filenames, want: `(?s)/(.*/|)à`},
	{pat: `/**foo`, mode: Filenames, want: `(?s)/.*foo`},
	{pat: `\*`, want: `\*`},
	{pat: `\`, wantErr: true},
	{pat: `?`, want: `(?s).`},
	{pat: `?`, mode: Filenames, want: `[^/]`},
	{pat: `?à`, want: `(?s).à`},
	{pat: `\a`, want: `a`},
	{pat: `(`, want: `\(`},
	{pat: `a|b`, want: `a\|b`},
	{pat: `x{3}`, want: `x\{3\}`},
	{pat: `{3,4}`, want: `\{3,4\}`},
	{pat: `{3,4}`, mode: Braces, want: `(?:3|4)`},
	{pat: `{3,`, want: `\{3,`},
	{pat: `{3,`, mode: Braces, want: `\{3,`},
	{pat: `{3,{4}`, mode: Braces, want: `\{3,\{4\}`},
	{pat: `{3,{4}}`, mode: Braces, want: `(?:3|\{4\})`},
	{pat: `{3,{4,[56]}}`, mode: Braces, want: `(?:3|(?:4|[56]))`},
	{pat: `{3..5}`, mode: Braces, want: `(?:3|4|5)`},
	{pat: `{9..12}`, mode: Braces, want: `(?:9|10|11|12)`},
	{pat: `[a]`, want: `[a]`},
	{pat: `[abc]`, want: `[abc]`},
	{pat: `[^bc]`, want: `[^bc]`},
	{pat: `[!bc]`, want: `[^bc]`},
	{pat: `[[]`, want: `[[]`},
	{pat: `[\]]`, want: `[\]]`},
	{pat: `[\]]`, mode: Filenames, want: `[\]]`},
	{pat: `[]]`, want: `[]]`},
	{pat: `[!]]`, want: `[^]]`},
	{pat: `[^]]`, want: `[^]]`},
	{pat: `[a/b]`, want: `[a/b]`},
	{pat: `[a/b]`, mode: Filenames, want: `\[a/b\]`},
	{pat: `[`, wantErr: true},
	{pat: `[\`, wantErr: true},
	{pat: `[^`, wantErr: true},
	{pat: `[!`, wantErr: true},
	{pat: `[!bc]`, want: `[^bc]`},
	{pat: `[]`, wantErr: true},
	{pat: `[^]`, wantErr: true},
	{pat: `[!]`, wantErr: true},
	{pat: `[ab`, wantErr: true},
	{pat: `[a-]`, want: `[a-]`},
	{pat: `[z-a]`, wantErr: true},
	{pat: `[a-a]`, want: `[a-a]`},
	{pat: `[aa]`, want: `[aa]`},
	{pat: `[0-4A-Z]`, want: `[0-4A-Z]`},
	{pat: `[-a]`, want: `[-a]`},
	{pat: `[^-a]`, want: `[^-a]`},
	{pat: `[a-]`, want: `[a-]`},
	{pat: `[[:digit:]]`, want: `[[:digit:]]`},
	{pat: `[[:`, wantErr: true},
	{pat: `[[:digit`, wantErr: true},
	{pat: `[[:wrong:]]`, wantErr: true},
	{pat: `[[=x=]]`, wantErr: true},
	{pat: `[[.x.]]`, wantErr: true},
}

func TestRegexp(t *testing.T) {
	t.Parallel()
	for i, tc := range translateTests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			got, gotErr := Regexp(tc.pat, tc.mode)
			if tc.wantErr && gotErr == nil {
				t.Fatalf("(%q, %b) did not error", tc.pat, tc.mode)
			}
			if !tc.wantErr && gotErr != nil {
				t.Fatalf("(%q, %b) errored with %q", tc.pat, tc.mode, gotErr)
			}
			if got != tc.want {
				t.Fatalf("(%q, %b) got %q, wanted %q", tc.pat, tc.mode, got, tc.want)
			}
			_, rxErr := syntax.Parse(got, syntax.Perl)
			if gotErr == nil && rxErr != nil {
				t.Fatalf("regexp/syntax.Parse(%q) failed with %q", got, rxErr)
			}
		})
	}
}
