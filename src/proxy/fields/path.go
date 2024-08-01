package fields

import (
	E "github.com/yusing/go-proxy/error"
	F "github.com/yusing/go-proxy/utils/functional"
)

type Path struct{ F.Stringable }

func NewPath(s string) (Path, E.NestedError) {
	if s == "" || s[0] == '/' {
		return Path{F.NewStringable(s)}, E.Nil()
	}
	return Path{}, E.Invalid("path", s).Extra("must be empty or start with '/'")
}
