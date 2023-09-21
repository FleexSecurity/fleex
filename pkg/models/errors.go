package models

import "errors"

var (
	ErrGeneric         = errors.New("something went wrong, check that the data in the config.yaml is correct")
	ErrInvalidProvider = errors.New("invalid provider")
	ErrInvalidToken    = errors.New("invalid token")
	ErrInvalidImage    = errors.New("invalid image")
	ErrInvalidRegion   = errors.New("invalid region")
	ErrInvalidSize     = errors.New("invalid size")
	ErrInvalidPort     = errors.New("invalid port")
	ErrInvalidSshFile  = errors.New("invalid SSH file")
	ErrBoxNotFound     = errors.New("box not found")
)
