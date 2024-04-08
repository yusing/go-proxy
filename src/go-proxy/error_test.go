package main

import (
	"testing"
)

func AssertEq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestErrorSimple(t *testing.T) {
	ne := NewNestedError("foo bar")
	AssertEq(t, ne.Error(), "foo bar")
	ne.Subject("baz")
	AssertEq(t, ne.Error(), "baz: foo bar")
}

func TestErrorSubjectOnly(t *testing.T) {
	ne := NewNestedError("").Subject("bar")
	AssertEq(t, ne.Error(), "bar")
}

func TestErrorExtra(t *testing.T) {
	ne := NewNestedError("foo").Extra("bar").Extra("baz")
	AssertEq(t, ne.Error(), "foo:\n  - bar\n  - baz\n")
}

func TestErrorNested(t *testing.T) {
	inner := NewNestedError("inner").
		Extra("123").
		Extra("456")
	inner2 := NewNestedError("inner").
		Subject("2").
		Extra("456").
		Extra("789")
	inner3 := NewNestedError("inner").
		Subject("3").
		Extra("456").
		Extra("789")
	ne := NewNestedError("foo").
		Extra("bar").
		Extra("baz").
		ExtraError(inner).
		With(inner.With(inner2.With(inner3)))
	want :=
		`foo:
  - bar
  - baz
  - inner:
    - 123
    - 456
  - inner:
    - 123
    - 456
    - 2: inner:
      - 456
      - 789
      - 3: inner:
        - 456
        - 789
`
	AssertEq(t, ne.Error(), want)
}
