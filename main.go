package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/ioutil"
	"unicode/utf16"
	"log"
	"flag"
)

type Single struct {
	Id uint32
	Str string

	str []uint16
}

type Index struct {
	Name string
	Single []Single

	off uint32
	name []uint16
}

type Ctx struct {
	// magic i think, i am not sure
	Unknow [8]byte
	Index []Index

	index_count uint32
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
	log.SetFlags(Llongfile)

	buf,e := ioutil.ReadFile(in)
	if e != nil {
		log.Println("failed to read input")
	}
	rd := bytes.NewReader(buf)

	switch mode {
	case "parse":
		e := binary.Read(rd, binary.LittleEndian, &h.Unknow)
		if e != nil {
			log.Println("failed to read")
		}

		e := binary.Read(rd, binary.LittleEndian, &h.index_count)
		if e != nil {
			log.Println("failed to read")
		}

		for a := uint32(0); a<h.index_count; a++ {
			l := uint32(0)
			e := binary.Read(rd, binary.LittleEndian, &l)
			if e != nil {
				log.Println("failed to read")
			}

			var i Index
			i.name = make([]uint16, l)
			e := binary.Read(rd, binary.LittleEndian, &i.name)
			if e != nil {
				log.Println("failed to read")
			}
			i.Name = string(utf16.Decode(i.name))

			e := binary.Read(rd, binary.LittleEndian, &i.off)
			if e != nil {
				log.Println("failed to read")
			}

			h.Index = append(h.Index, i)
		}
		h.header_size = rd.Size() - int64(rd.Len())

		off := uint32(0)
		for n,_ := range h.Index {
			i := &h.Index[n]
			e := rd.Seek(h.header_size+int64(i.off), io.SeekStart)
			if e != nil {
				log.Println("failed to seek", h.header_size+int64(i.off))
			}

			var end uint32
			if (n+1) < len(h.Index) {
				end = h.Index[n+1].off
			} else {
				end = 0
			}

			for {
				var s Single
				e := binary.Read(rd, binary.LittleEndian, &s.Id)
				if e == io.EOF {
					break
				} else if e != nil {
					log.Println("failed to read")
				}

				l := uint32(0)
				e := binary.Read(rd, binary.LittleEndian, &l)
				if e != nil {
					log.Println("failed to read")
				}
				if l != 0 {
					s.str = make([]uint16, l)
					e := binary.Read(rd, binary.LittleEndian, &s.str)
					if e != nil {
						log.Println("failed to read")
					}
					s.Str = string(utf16.Decode(s.str))
				}
				off += l*2+8

				i.Single = append(i.Single, s)
				if off == end {
					break
				}
			}
		}

		encoder := json.NewEncoder(&buffer)
		encoder.SetIndent("", "\t")
		e := encoder.Encode(&h)
		if e != nil {
			log.Println("failed to convert to json")
		}
	case "compile":
		e := json.NewDecoder(rd).Decode(&h)
		if e != nil {
			log.Println("failed to parse json")
		}

		off := uint32(0)
		h.header_size += int64(len(h.Index)*(4*2))
		h.index_count = uint32(len(h.Index))
		for _,i := range h.Index {
			h.header_size += int64(len(i.name))
		}

		for n,_ := range h.Index {
			i := &h.Index[n]
			i.name = utf16.Encode([]rune(i.Name))
			i.off = off

			for n,_ := range i.Single {
				s := &i.Single[n]
				s.str = utf16.Encode([]rune(s.Str))
				off += uint32(len(s.str)*2)+(4*2)
			}
		}

		e := binary.Write(&buffer, binary.LittleEndian, h.Unknow)
		if e != nil {
			log.Println("failed to write")
		}
		e := binary.Write(&buffer, binary.LittleEndian, h.index_count)
		if e != nil {
			log.Println("failed to write")
		}
		for _,i := range h.Index {
			e := binary.Write(&buffer, binary.LittleEndian, uint32(len(i.name)))
			if e != nil {
				log.Println("failed to write")
			}
			e := binary.Write(&buffer, binary.LittleEndian, i.name)
			if e != nil {
				log.Println("failed to write")
			}
			e := binary.Write(&buffer, binary.LittleEndian, i.off)
			if e != nil {
				log.Println("failed to write")
			}
		}

		for _,i := range h.Index {
			for _,s := range i.Single {
				e := binary.Write(&buffer, binary.LittleEndian, s.Id)
				if e != nil {
					log.Println("failed to write")
				}
				e := binary.Write(&buffer, binary.LittleEndian, uint32(len(s.str)))
				if e != nil {
					log.Println("failed to write")
				}
				e := binary.Write(&buffer, binary.LittleEndian, s.str)
				if e != nil {
					log.Println("failed to write")
				}
			}
		}
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Println("failed to write output")
	}
}
