package main

import "unicode/utf16"

type String struct {
	L uint32
	R []uint16 `length:"R_length"`
}

func (s *String) Unmarshal() string {
	return string(utf16.Decode(s.R))
}

func (s *String) Marshal(t string) {
	s.R = utf16.Encode([]rune(t))
	s.L = uint32(len(s.R))
}

type StringWithID struct {
	ID   uint64
	RStr String `json:"-"`
	Str  string `skip:"rw"`
}

func (s *StringWithID) Unmarshal() {
	s.Str = s.RStr.Unmarshal()
}

func (s *StringWithID) Marshal() {
	s.RStr.Marshal(s.Str)
}

