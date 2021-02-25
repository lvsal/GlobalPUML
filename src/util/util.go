package util

import (
	"os"
	"fmt"
	"strings"
	//"unicode"
	"regexp"
)

var Debug bool

func PrintErr(err error) {
	fmt.Fprintf(os.Stderr, err.Error() + "\n")
}

// Remove comments.
// Source: https://rosettacode.org/wiki/Strip_comments_from_a_string#Go
const (
	commentChars = "#;"
	singleLineRegex = "//.*?\n"
	multiLineRegex = "/\\*(.|\\s)*?\\*/"
)
func StripComment(source string) string {
	/*
	if cut := strings.IndexAny(source, commentChars); cut >= 0 {
		source = strings.TrimRightFunc(source[:cut], unicode.IsSpace)
	}
	*/

	
	re := regexp.MustCompile(singleLineRegex)
	source = re.ReplaceAllString(source, "\n")

	re = regexp.MustCompile(multiLineRegex)
	return re.ReplaceAllString(source, "\n")
}
///////////////////////////////////////////////////////////////////////

// Substitute strings.ReplaceAll for Go v1.10.
func ReplaceAll(s, old, new string) string {
	n := strings.Count(s, old)
	return strings.Replace(s, old, new, n)
}