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

	"github.com/dop251/goja"
	"github.com/xhebox/bstruct"
	"golang.org/x/text/encoding/unicode"
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
			Len   uint32   `json:"-" prog:"if (!proc&&prog) cur.Len.set(cur.Rname.value().length)"`
			Name  []uint16 `json:"-" length:"cur.Len.value()" skip:"w"`
			Rname string   `prog:"if (proc&&prog) cur.Rname.set(utf16to8(cur.Name.value())); if (!proc&&prog) cur.Rname.set(utf8to16(cur.Rname.value()));"`
			Off   uint32
		} `length:"idxcnt=cur.Index_count.value(); idxcnt"`
	} `prog:"hdr = cur.Hdr; if (proc&&!prog) read(hdr.Indexs[0].Off.value());"`
	Body []struct {
		Singles []struct {
			Id   uint32
			Len  uint32   `json:"-" prog:"if (!proc&&prog) cur.Len.set(cur.Rstr.value().length)"`
			Str  []uint16 `json:"-" length:"cur.Len.value()" skip:"w"`
			Rstr string   `prog:"if (proc&&prog) cur.Rstr.set(utf16to8(cur.Str.value())); if (!proc&&prog) cur.Rstr.set(utf8to16(cur.Rstr.value()));"`
		} `size:"kstack[0] < idxcnt-1 ? hdr.Indexs[kstack[0]+1].Off.value() - hdr.Indexs[kstack[0]].Off.value() : -1"`
	} `length:"idxcnt"`
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

	t, e := bstruct.New(h)
	if e != nil {
		log.Fatalln(e)
	}

	rt := goja.New()
	rt.Set("utf16to8", func(f goja.FunctionCall) goja.Value {
		if len(f.Arguments) != 1 {
			panic("except a slice of uint16")
		}

		v := f.Arguments[0].Export()
		if rv, ok := v.([]uint16); ok {
			return rt.ToValue(string(utf16.Decode(rv)))
		}

		return rt.ToValue("")
	})
	rt.Set("utf8to16", func(f goja.FunctionCall) goja.Value {
		if len(f.Arguments) != 1 {
			panic("except one argument")
		}

		var src string
		srcv := f.Arguments[0].Export()
		src, ok := srcv.(string)
		if !ok {
			panic("except a string")
		}

		str, e := unicode.All[3].NewEncoder().String(src)
		if e != nil {
			panic(e)
		}

		return rt.ToValue(str)
	})

	switch mode {
	case "parse":
		if e := t.Read(rd, &h, rt); e != nil {
			log.Fatalf("%+v\n", e)
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

		h.Hdr.Index_count = uint32(len(h.Body))
		if h.Hdr.Index_count != uint32(len(h.Hdr.Indexs)) {
			log.Fatalln("index is not consistent with body")
		}

		// update Off
		h.Hdr.Indexs[0].Off = 0
		for i, v := range h.Body[:len(h.Body)-1] {
			total := uint32(0)
			for _, k := range v.Singles {
				total += uint32(utf8.RuneCountInString(k.Rstr))
			}
			h.Hdr.Indexs[i+1].Off = total
		}

		if e := t.Write(&buffer, &h, rt); e != nil {
			log.Fatalf("%+v\n", e)
		}
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Fatalln("failed to write output")
	}

	fmt.Printf("x %+v\n", h.Hdr)
}
