package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/xhebox/bstruct"
)

type Ctx_Index struct {
	Len   uint32   `json:"-"`
	Rname []uint16 `json:"-" length:"Rname_length"`
	Name  string   `skip:"rw"`
	Off   uint32
}

type Ctx_Single struct {
	Id   uint32
	Len  uint32   `json:"-"`
	Rstr []uint16 `json:"-" length:"Rstr_length"`
	Str  string   `skip:"rw"`
}

type Ctx struct {
	// magic i think, i am not sure
	Unknow      [8]byte
	Index_count uint32
	Indexs      []Ctx_Index `length:"Indexs_length"`
	Body        []struct {
		Singles []Ctx_Single `size:"singles_size"`
	} `length:"body_length" rdm:"body_rdm"`
}

func main() {
	var in, out, mode string
	var buffer bytes.Buffer
	var h Ctx
	flag.StringVar(&in, "i", "input", "input file")
	flag.StringVar(&out, "o", "output", "output file")
	flag.StringVar(&mode, "m", "parse", "could be parse(ctx to json)/compile(json to ctx)")
	flag.Parse()
	log.SetFlags(log.Llongfile)

	buf, e := ioutil.ReadFile(in)
	if e != nil {
		log.Fatalln("failed to read input")
	}
	rd := bytes.NewReader(buf)

	t := bstruct.MustNew(h)

	switch mode {
	case "parse":
		dec := bstruct.NewDecoder()
		dec.Rd = rd

		dec.Runner.Register("Rname_length", func(s ...interface{}) interface{} {
			return int(s[1].(Ctx_Index).Len)
		})

		dec.Runner.Register("Rstr_length", func(s ...interface{}) interface{} {
			return int(s[1].(Ctx_Single).Len)
		})

		dec.Runner.Register("body_length", func(s ...interface{}) interface{} {
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

			if body_count < int(r.Index_count-1) {
				body_count++
				return int(r.Indexs[body_count+1].Off - r.Indexs[body_count].Off)
			} else {
				body_count++
				return -1
			}
		})

		if e := dec.Decode(t, &h); e != nil {
			log.Fatalf("%+v\n", e)
		}

		idxs := h.Indexs
		for k := range idxs {
			idxs[k].Name = string(utf16.Decode(idxs[k].Rname))
		}

		for _, body := range h.Body {
			singles := body.Singles
			for k := range singles {
				singles[k].Str = string(utf16.Decode(singles[k].Rstr))
			}
		}

		encoder := json.NewEncoder(&buffer)
		encoder.SetIndent("", "\t")
		e = encoder.Encode(&h)
		if e != nil {
			log.Fatalln("failed to convert to json")
		}
	case "compile":
		e = json.NewDecoder(rd).Decode(&h)
		if e != nil {
			log.Fatalln("failed to parse json")
		}

		idxs := h.Indexs
		for k := range idxs {
			idxs[k].Rname = utf16.Encode([]rune(idxs[k].Name))
		}
		h.Index_count = uint32(len(idxs))

		if h.Index_count != uint32(len(h.Body)) {
			log.Fatalln("index is not consistent with body")
		}

		h.Indexs[0].Off = 0
		for i, body := range h.Body {
			total := uint32(0)
			check := i < len(h.Body)-1

			singles := body.Singles
			for k := range singles {
				singles[k].Rstr = utf16.Encode([]rune(singles[k].Str))
				if check {
					total += uint32(utf8.RuneCountInString(singles[k].Str))
				}
			}

			if check {
				h.Indexs[i+1].Off = total
			}
		}

		enc := bstruct.NewEncoder()
		enc.Wt = &buffer

		if e := enc.Encode(t, &h); e != nil {
			log.Fatalf("%+v\n", e)
		}
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Fatalln("failed to write output")
	}
}
