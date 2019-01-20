package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"github.com/xhebox/bstruct"
)

func str_bytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr := reflect.SliceHeader{Data: sh.Data, Len: sh.Len, Cap: sh.Len}
	return *(*[]byte)(unsafe.Pointer(&hdr))
}

type Ctx struct {
	Hdr struct {
		// magic i think, i am not sure
		Unknow      [8]byte
		Index_count uint32
		Indexs      []struct {
			Len   uint32   `json:"-"`
			Rname []uint16 `json:"-" length:"current.Len"`
			Name  string   `skip:"rw"`
			Off   uint32
		} `length:"root.Hdr.Index_count"`
	} `rdn:"read(root.Hdr.Indexs[0].Off)"`
	Body []struct {
		Singles []struct {
			Id   uint32
			Len  uint32   `json:"-"`
			Rstr []uint16 `json:"-" length:"current.Len"`
			Str  string   `skip:"rw"`
		} `size:"k < root.Hdr.Index_count-1 ? root.Hdr.Indexs[k+1].Off - root.Hdr.Indexs[k].Off : -1"`
	} `length:"root.Hdr.Index_count"`
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

		if e := dec.Decode(t, &h); e != nil {
			log.Fatalf("%+v\n", e)
		}

		idxs := h.Hdr.Indexs
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

		idxs := h.Hdr.Indexs
		for k := range idxs {
			idxs[k].Rname = utf16.Encode([]rune(idxs[k].Name))
		}
		h.Hdr.Index_count = uint32(len(idxs))

		if h.Hdr.Index_count != uint32(len(h.Body)) {
			log.Fatalln("index is not consistent with body")
		}

		h.Hdr.Indexs[0].Off = 0
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
				h.Hdr.Indexs[i+1].Off = total
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

	fmt.Printf("x %+v\n", h.Hdr)
}
