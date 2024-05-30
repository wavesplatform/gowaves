package main

type input int

//go:generate stringer -type=input -trimprefix input
const (
	inputJSON input = iota
	inputBinary
)
