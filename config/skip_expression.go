package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-andiamo/splitter"
	// "github.com/romnn/kube-score/domain"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

func unquote(value string) string {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return value
	}
	return unquoted
}

type SkipExpression struct {
	RawPath    string
	Path       *yamlpath.Path
	RawValue   string
	ValueRegex *regexp.Regexp
}

func ParseSkipExpression(rawExpression string) (*SkipExpression, error) {
	rawPath, value, err := splitRawExpression(rawExpression)
	if err != nil {
		return nil, err
	}
	rawPath = unquote(rawPath)
	value = unquote(value)

	// fmt.Printf("skip expression:\n")
	// fmt.Printf("\traw   = %q\n", rawExpression)
	// fmt.Printf("\tpath  = %q\n", rawPath)
	// fmt.Printf("\tvalue = %q\n", value)

	path, err := yamlpath.NewPath(rawPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", rawPath, err)
	}

	// fmt.Printf("parsed path=%+v\n", path)

	valueRegex, err := regexp.Compile(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value pattern %q: %w", valueRegex.String(), err)
	}

	expr := &SkipExpression{
		RawPath:    rawPath,
		Path:       path,
		RawValue:   value,
		ValueRegex: valueRegex,
	}
	return expr, nil
}

func (e *SkipExpression) String() string {
	return fmt.Sprintf("%s=%s", e.RawPath, e.RawValue)
}

func (e *SkipExpression) Evaluate(doc yaml.Node) bool {
	// func (e *SkipExpression) Evaluate(value any) bool {
	// to yaml
	// yaml.Marshal(in interface{})

	matches, err := e.Path.Find(&doc)
	if err != nil {
		return false
	}

	if len(matches) < 1 {
		return false
	}

	for _, match := range matches {
		value := strings.TrimSpace(match.Value)
		// logger.Debug("match", zap.String("path", e.RawPath), zap.String("value", value))
		if !e.ValueRegex.Match([]byte(value)) {
			return false
		}
	}

	return true
}

func splitRawExpression(value string) (string, string, error) {
	equalSplitter := splitter.MustCreateSplitter('=', splitter.SingleQuotes)
	parts, err := equalSplitter.Split(value)
	if err != nil {
		return "", "", err
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid expression %q", value)
	}
	return parts[0], parts[1], nil
}
