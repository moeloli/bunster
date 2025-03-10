package dottest_test

import (
	"reflect"
	"testing"

	"github.com/yassinebenaid/bunster/pkg/dottest"
	"github.com/yassinebenaid/godump"
)

var dump = (&godump.Dumper{
	Theme:                   godump.DefaultTheme,
	ShowPrimitiveNamedTypes: true,
}).Sprintln

func TestParser_Parse(t *testing.T) {
	testCases := []struct {
		input    string
		expected []dottest.Test
		err      string
	}{
		{
			input: `
#(TEST: foo bar)
#(RESULT)
#(ENDTEST)`,
			expected: []dottest.Test{{
				Label: `foo bar`,
			}},
		},
		{
			input: `


#(TEST: foo bar)

foo bar
 
#(RESULT)
 
foo bar
 
#(ENDTEST)


`,
			expected: []dottest.Test{{
				Label:  `foo bar`,
				Input:  "\nfoo bar\n \n",
				Output: " \nfoo bar\n \n",
			}},
		},
		{
			input: `


#(TEST: test 1)
input-1
#(RESULT)
output-1
#(ENDTEST)

#(TEST: test 2)
input-2
#(RESULT)
output-2
#(ENDTEST)

`,
			expected: []dottest.Test{
				{Label: `test 1`, Input: "input-1\n", Output: "output-1\n"},
				{Label: `test 2`, Input: "input-2\n", Output: "output-2\n"},
			},
		},
	}

	for i, tc := range testCases {
		tests, parseErr := dottest.Parse(tc.input)

		if tc.err != "" {
			if parseErr == nil || parseErr.Error() != tc.err {
				t.Errorf("expected:\n%sgot:\n%s", dump(tc.err), dump(parseErr))
			}
			return
		}

		if parseErr != nil {
			t.Errorf("unexpected error: %v", parseErr)
			return
		}

		if !reflect.DeepEqual(tc.expected, tests) {
			t.Errorf("Case: %d\nExpected:\n%sGot:\n%s", i, dump(tc.expected), dump(tests))
		}
	}
}

func TestParseErrors(t *testing.T) {
	testCases := []struct {
		input string
		err   string
	}{
		{
			input: `foo bar`,
			err:   `line 1: bad test syntax, coundl't find test header '#(TEST: ...)', found "foo bar"`,
		},
		{
			input: `#(TEST: foo bar `,
			err:   `line 1: bad test syntax, unclosed test header '#(TEST: ...)'`,
		},
		{
			input: `#(TEST: foo bar)`,
			err:   "bad test syntax, coundl't find #(RESULT) section",
		},
		{
			input: `#(TEST: foo bar) extra-string`,
			err:   "line 1: bad test syntax, unclosed test header '#(TEST: ...)'",
		},
		{
			input: "#(TEST: foo bar) \n#(RESULT)",
			err:   "bad test syntax, unclosed test, missing '#(ENDTEST)'",
		},
		{
			input: "#(TEST: ) \n#(RESULT)\n#(ENDTEST)",
			err:   "line 1: bad test syntax, test label cannot be blank",
		},
		{
			input: "#(TEST: foobar )\n#(TEST: foobar )\n#(RESULT)\n#(ENDTEST)",
			err:   "line 2: bad test syntax, coundl't find #(RESULT) section",
		},
		{
			input: "#(TEST: foobar )\n#(ENDTEST)",
			err:   "line 2: bad test syntax, coundl't find #(RESULT) section",
		},
		{
			input: "#(TEST: foobar )\n#(RESULT)\n#(RESULT)\n#(ENDTEST)",
			err:   "line 3: bad test syntax, unclosed test, missing '#(ENDTEST)'",
		},
		{
			input: "#(TEST: foobar )\n#(RESULT)\n#(TEST: foo)\n#(ENDTEST)",
			err:   "line 3: bad test syntax, unclosed test, missing '#(ENDTEST)'",
		},
	}

	for _, tc := range testCases {
		_, parseErr := dottest.Parse(tc.input)

		if parseErr == nil || parseErr.Error() != tc.err {
			t.Errorf("expected:\n%sgot:\n%s", dump(tc.err), dump(parseErr))
		}
	}
}
