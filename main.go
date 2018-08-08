package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"unicode/utf16"
)

type Single struct {
	Id  uint32

	// this is the original data
	str []uint16

	// this is the decoded string, not really included by ctx file
	Str string
}

type Index struct {
	name []uint16
	off  uint32

	// this should be the body of ctx file, but moved here to transform conveniently between json/ctx
	Name   string
	Singles []Single
}

type Ctx struct {
	// magic i think, i am not sure
	Unknow [8]byte
	index_count uint32
	Indexs  []Index
	header_size int64
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

	switch mode {
	case "parse":
		e = binary.Read(rd, binary.LittleEndian, &h.Unknow)
		if e != nil {
			log.Fatalln("failed to read Header.Unknow")
		}

		e = binary.Read(rd, binary.LittleEndian, &h.index_count)
		if e != nil {
			log.Fatalln("failed to read Header.index_count")
		}

		for a := uint32(0); a < h.index_count; a++ {
			var i Index

			l := uint32(0)
			e = binary.Read(rd, binary.LittleEndian, &l)
			if e != nil {
				log.Fatalln("failed to read the length of Header.Index[", a, " ].name")
			}

			i.name = make([]uint16, l)
			e = binary.Read(rd, binary.LittleEndian, &i.name)
			if e != nil {
				log.Fatalln("failed to read Header.Index[", a, " ].name")
			}
			i.Name = string(utf16.Decode(i.name))

			e = binary.Read(rd, binary.LittleEndian, &i.off)
			if e != nil {
				log.Fatalln("failed to read Header.Index[", a, " ].off")
			}

			h.Indexs = append(h.Indexs, i)
		}
		h.header_size = rd.Size() - int64(rd.Len())

		off := uint32(0)
		for n, _ := range h.Indexs {
			i := &h.Indexs[n]
			_, e = rd.Seek(h.header_size+int64(i.off), io.SeekStart)
			if e != nil {
				log.Fatalln("failed to seek Body[", n, "], ", h.header_size+int64(i.off))
			}

			var end uint32
			if (n + 1) < len(h.Indexs) {
				end = h.Indexs[n+1].off
			} else {
				end = 0
			}

			cnt := uint32(0)
			for {
				var s Single
				e = binary.Read(rd, binary.LittleEndian, &s.Id)
				if e == io.EOF {
					break
				} else if e != nil {
					log.Fatalln("failed to read Body[", n, "].Singles[", cnt, "].Id")
				}

				l := uint32(0)
				e = binary.Read(rd, binary.LittleEndian, &l)
				if e != nil {
					log.Fatalln("failed to read the len of Body[", n, "].Singles[", cnt, "].Str")
				}

				if l != 0 {
					s.str = make([]uint16, l)
					e = binary.Read(rd, binary.LittleEndian, &s.str)
					if e != nil {
						log.Fatalln("failed to read Body[", n, "].Singles[", cnt, "].Str")
					}

					s.Str = string(utf16.Decode(s.str))
				}
				off += l*2 + 8

				cnt++
				i.Singles = append(i.Singles, s)
				if off == end {
					break
				}
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

		off := uint32(0)
		h.index_count = uint32(len(h.Indexs))
		h.header_size += int64(len(h.Indexs) * (4 * 2))
		for _, i := range h.Indexs {
			h.header_size += int64(len(i.name))
		}

		for n, _ := range h.Indexs {
			i := &h.Indexs[n]
			i.name = utf16.Encode([]rune(i.Name))
			i.off = off

			for n, _ := range i.Singles {
				s := &i.Singles[n]
				s.str = utf16.Encode([]rune(s.Str))
				off += uint32(len(s.str)*2) + (4 * 2)
			}
		}

		e = binary.Write(&buffer, binary.LittleEndian, h.Unknow)
		if e != nil {
			log.Fatalln("failed to write Header.Unknown")
		}

		e = binary.Write(&buffer, binary.LittleEndian, h.index_count)
		if e != nil {
			log.Fatalln("failed to write Header.index_count")
		}

		for n, i := range h.Indexs {
			e = binary.Write(&buffer, binary.LittleEndian, uint32(len(i.name)))
			if e != nil {
				log.Fatalln("failed to write the len of Header.Index[", n, "].name")
			}

			e = binary.Write(&buffer, binary.LittleEndian, i.name)
			if e != nil {
				log.Fatalln("failed to write Header.Index[", n, "].name")
			}

			e = binary.Write(&buffer, binary.LittleEndian, i.off)
			if e != nil {
				log.Fatalln("failed to write Header.Index[", n, "].off")
			}
		}

		for n, i := range h.Indexs {
			for c, s := range i.Singles {
				e = binary.Write(&buffer, binary.LittleEndian, s.Id)
				if e != nil {
					log.Fatalln("failed to write Body.Index[", n, "].Singles[", c, "].Id")
				}

				e = binary.Write(&buffer, binary.LittleEndian, uint32(len(s.str)))
				if e != nil {
					log.Fatalln("failed to write the len of Body.Index[", n, "].Singles[", c, "].str")
				}

				e = binary.Write(&buffer, binary.LittleEndian, s.str)
				if e != nil {
					log.Fatalln("failed to write Body.Index[", n, "].Singles[", c, "].str")
				}
			}
		}
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Fatalln("failed to write output")
	}
}
