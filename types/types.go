package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Metadata is an ADT to overcome the generic repo problem with JSONB Value
type Metadata map[string]interface{}

// Value override value's function for metadata (ADT) type
func (p Metadata) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

// Scan override scan's function for metadata (ADT) type
func (p *Metadata) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	if string(source) == "{}" || string(source) == "null" {
		*p = map[string]interface{}{}
		return nil
	}

	*p, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("Type assertion .(map[string]interface{}) failed")
	}

	return nil
}

// IntArray is an ADT to overcome the generic repo problem with pq.StringArray Value
type IntArray []int

// Value override value's function for IntArray (ADT) type
func (a IntArray) Value() (driver.Value, error) {
	var strs []string
	for _, i := range a {
		strs = append(strs, strconv.FormatInt(int64(i), 10))
	}
	return "{" + strings.Join(strs, ",") + "}", nil
}

// Scan override scan's function for IntArray (ADT) type
func (a *IntArray) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	asString := string(asBytes)
	parsed, err := parseArrayInt(asString)
	if err != nil {
		return err
	}
	(*a) = IntArray(parsed)

	return nil
}

// StringArray is an ADT to overcome the generic repo problem with pq.StringArray Value
type StringArray []string

// Value override value's function for StringArray (ADT) type
func (s StringArray) Value() (driver.Value, error) {
	for i, elem := range s {
		s[i] = `"` + strings.Replace(strings.Replace(elem, `\`, `\\\`, -1), `"`, `\"`, -1) + `"`
	}
	return "{" + strings.Join(s, ",") + "}", nil
}

// Scan override scan's function for StringArray (ADT) type
func (s *StringArray) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	asString := string(asBytes)
	parsed := parseArrayString(asString)
	(*s) = StringArray(parsed)

	return nil
}

// construct a regexp to extract values:
var (
	// unquoted array values must not contain: (" , \ { } whitespace NULL)
	// and must be at least one char
	unquotedChar  = `[^",\\{}\s(NULL)]`
	unquotedValue = fmt.Sprintf("(%s)+", unquotedChar)

	// quoted array values are surrounded by double quotes, can be any
	// character except " or \, which must be backslash escaped:
	quotedChar  = `[^"\\]|\\"|\\\\`
	quotedValue = fmt.Sprintf("\"(%s)*\"", quotedChar)

	// an array value may be either quoted or unquoted:
	arrayValue = fmt.Sprintf("(?P<value>(%s|%s))", unquotedValue, quotedValue)

	// Array values are separated with a comma IF there is more than one value:
	arrayExp = regexp.MustCompile(fmt.Sprintf("%s", arrayValue))

	valueIndex int
)

// Parse the output string from the array type.
// Regex used: (((?P<value>(([^",\\{}\s(NULL)])+|"([^"\\]|\\"|\\\\)*")))(,)?)
func parseArrayString(array string) []string {
	results := make([]string, 0)
	matches := arrayExp.FindAllStringSubmatch(array, -1)
	for _, match := range matches {
		s := match[valueIndex]
		// the string _might_ be wrapped in quotes, so trim them:
		s = strings.Trim(s, "\"")
		results = append(results, s)
	}
	return results
}

// Parse the output int from the array type.
// Regex used: (((?P<value>(([^",\\{}\s(NULL)])+|"([^"\\]|\\"|\\\\)*")))(,)?)
func parseArrayInt(array string) ([]int, error) {
	results := make([]int, 0)
	matches := arrayExp.FindAllStringSubmatch(array, -1)
	for _, match := range matches {
		s := match[valueIndex]
		// the string _might_ be wrapped in quotes, so trim them:
		s = strings.Trim(s, "\"")
		sInt, err := strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
		results = append(results, sInt)
	}
	return results, nil
}
