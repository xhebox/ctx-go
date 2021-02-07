package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/xhebox/bstruct"
)

type CtxIndex struct {
	Str  String `json:"-"`
	Name string `skip:"rw"`
	Off  uint32
}

func (i *CtxIndex) Unmarshal() {
	i.Name = i.Str.Unmarshal()
}

func (i *CtxIndex) Marshal() {
	i.Str.Marshal(i.Name)
}

type CtxSingle struct {
	Id   uint32
	Rstr String `json:"-"`
	Str  string `skip:"rw"`
}

func (i *CtxSingle) Unmarshal() {
	i.Str = i.Rstr.Unmarshal()
}

func (i *CtxSingle) Marshal() {
	i.Rstr.Marshal(i.Str)
}

type Ctx struct {
	// magic i think, i am not sure
	Unknow      [8]byte
	Index_count uint32     `json:"-"`
	Indexs      []CtxIndex `length:"Index_count"`
	Body        []struct {
		Singles []CtxSingle `size:"singles_size"`
	} `length:"Index_count" rdm:"body_rdm"`
}

func (i *Ctx) Unmarshal() {
	fmt.Printf("%+v\n", i)
	for k := range i.Indexs {
		i.Indexs[k].Unmarshal()
	}

	for j := range i.Body {
		body := &i.Body[j]
		for k := range body.Singles {
			body.Singles[k].Unmarshal()
		}
	}
}

func (h *Ctx) Marshal() error {
	h.Index_count = uint32(len(h.Indexs))
	for k := range h.Indexs {
		h.Indexs[k].Marshal()
	}

	if h.Index_count != uint32(len(h.Body)) {
		return fmt.Errorf("index is not consistent with body")
	}

	h.Indexs[0].Off = 0
	total := uint32(0)
	for i := range h.Body {
		body := &h.Body[i]

		check := i < len(h.Body)-1

		singles := body.Singles
		for k := range singles {
			singles[k].Marshal()
			if check {
				total += 8 + singles[k].Rstr.L*2
			}
		}

		if check {
			h.Indexs[i+1].Off = total
		}
	}

	return nil
}

func parseCtx(rd io.Reader, wt io.Writer) error {
	var h Ctx

	ctxType := bstruct.MustNew(h)

	dec := bstruct.NewDecoder()
	dec.Rd = rd

	dec.Runner.Register("R_length", func(s ...interface{}) interface{} {
		return int(s[1].(String).L)
	})

	dec.Runner.Register("Index_count", func(s ...interface{}) interface{} {
		return int(s[0].(*Ctx).Index_count)
	})

	dec.Runner.Register("body_rdm", func(s ...interface{}) interface{} {
		r := s[0].(*Ctx)
		buf := make([]byte, r.Indexs[0].Off)
		dec.Rd.Read(buf)
		return nil
	})

	body_count := 0
	dec.Runner.Register("singles_size", func(s ...interface{}) interface{} {
		r := s[0].(*Ctx)

		body_count++
		if body_count < int(r.Index_count) {
			return int(r.Indexs[body_count].Off - r.Indexs[body_count-1].Off)
		} else {
			return -1
		}
	})

	if e := dec.Decode(ctxType, &h); e != nil {
		return e
	}

	h.Unmarshal()

	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(h)
}

func compileCtx(rd io.Reader, wt io.Writer) error {
	var h Ctx

	ctxType := bstruct.MustNew(h)

	e := json.NewDecoder(rd).Decode(&h)
	if e != nil {
		return e
	}

	if e := h.Marshal(); e != nil {
		return e
	}

	enc := bstruct.NewEncoder()
	enc.Wt = wt

	return enc.Encode(ctxType, &h)
}
