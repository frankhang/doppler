package main

import (
	"github.com/frankhang/util/errors"
)

// Error code.
const (
	CodeBadFormat errors.ErrCode = iota + 1
	CodeMissingDelim
	CodeNoMetricType
	CodeBadMetricType

)

// Error classes.
const (

)

// Global error instances.
var (
	ErrBadFormat     = errors.ClassClient.New(CodeNoMetricType, "format: bad fmt [%s]")
	ErrMissingDelim  = errors.ClassClient.New(CodeMissingDelim, "format: missing delimeter: %c for %s")
	ErrNoMetricType  = errors.ClassClient.New(CodeNoMetricType, "format: no metric type")
	ErrBadMetricType = errors.ClassClient.New(CodeBadMetricType, "format: bad metric type: %c")
)

var errClz2Str = map[errors.ErrClass]string{

}

func init() {
	for k, v := range errors.ErrClz2Str {
		errClz2Str[k] = v
	}
}
