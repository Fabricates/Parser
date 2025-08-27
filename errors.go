package parser

import "errors"

// Common errors
var (
	ErrTemplateNotFound = errors.New("template not found")
	ErrWatcherClosed    = errors.New("file watcher is closed")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrParserClosed     = errors.New("parser is closed")
)
