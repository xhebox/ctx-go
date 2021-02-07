package main

import (
	"encoding/json"
	"io"

	"github.com/xhebox/bstruct"
)

type GlossaryItem struct {
	Target StringWithID
	RStr   String `json:"-"`
	Str    string `skip:"rw"`
	Un1    uint32
}

func (s *GlossaryItem) Unmarshal() {
	s.Target.Unmarshal()
	s.Str = s.RStr.Unmarshal()
}

func (s *GlossaryItem) Marshal() {
	s.Target.Marshal()
	s.RStr.Marshal(s.Str)
}

type GlossarySet struct {
	Category StringWithID
	Len      uint32         `json:"-"`
	Items    []GlossaryItem `length:"Items_length"`
}

func (item *GlossarySet) Unmarshal() {
	item.Category.Unmarshal()
	for i := range item.Items {
		item.Items[i].Unmarshal()
	}
}

func (item *GlossarySet) Marshal() {
	item.Category.Marshal()
	item.Len = uint32(len(item.Items))
	for i := range item.Items {
		item.Items[i].Marshal()
	}
}

type Glossary struct {
	Len        uint32        `json:"-"`
	Glossaries []GlossarySet `length:"Sets_length"`
}

func (q *Glossary) Unmarshal() {
	for i := range q.Glossaries {
		q.Glossaries[i].Unmarshal()
	}
}

func (q *Glossary) Marshal() {
	q.Len = uint32(len(q.Glossaries))
	for i := range q.Glossaries {
		q.Glossaries[i].Marshal()
	}
}

func parseGlossary(rd io.Reader, wt io.Writer) error {
	var h Glossary

	ctxType := bstruct.MustNew(h)

	dec := bstruct.NewDecoder()
	dec.Rd = rd

	dec.Runner.Register("R_length", func(s ...interface{}) interface{} {
		return int(s[1].(String).L)
	})

	dec.Runner.Register("Items_length", func(s ...interface{}) interface{} {
		return int(s[1].(GlossarySet).Len)
	})

	dec.Runner.Register("Sets_length", func(s ...interface{}) interface{} {
		return int(s[1].(Glossary).Len)
	})

	if e := dec.Decode(ctxType, &h); e != nil {
		return e
	}

	h.Unmarshal()
	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&h)
}

func compileGlossary(rd io.Reader, wt io.Writer) error {
	var h Glossary

	ctxType := bstruct.MustNew(h)

	e := json.NewDecoder(rd).Decode(&h)
	if e != nil {
		return e
	}

	h.Marshal()
	enc := bstruct.NewEncoder()
	enc.Wt = wt
	return enc.Encode(ctxType, &h)
}
