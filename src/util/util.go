package util

import (
	"bytes"
	"os"
	"fmt"
	"strings"
	//"unicode"
	"regexp"
	"encoding/json"
)

var Debug, Global bool

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

func Dump(v interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}