package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
)

type Coder = func(rd io.Reader, wt io.Writer) error

var (
	encoders = map[string]Coder{
		"quests":   compileQuests,
		"ctx":      compileCtx,
		"glossary": compileGlossary,
	}
	decoders = map[string]Coder{
		"quests":   parseQuests,
		"ctx":      parseCtx,
		"glossary": parseGlossary,
	}
)

func main() {
	var in, out, mode, format string
	var buffer bytes.Buffer
	flag.StringVar(&in, "i", "input", "input file")
	flag.StringVar(&out, "o", "output", "output file")
	flag.StringVar(&format, "f", "ctx", "ctx, quests, dialogues, glossary")
	flag.StringVar(&mode, "m", "parse", "could be parse(to json)/compile(from json)")
	flag.Parse()
	log.SetFlags(log.Llongfile)

	buf, e := ioutil.ReadFile(in)
	if e != nil {
		log.Fatalln("failed to read input")
	}
	rd := bytes.NewReader(buf)

	switch mode {
	case "parse":
		decoder, ok := decoders[format]
		if !ok {
			log.Fatalf("unsupport format: %s\n", format)
		}

		if e := decoder(rd, &buffer); e != nil {
			log.Fatalf("failed to convert %s to json: %s\n", format, e)
		}
	case "compile":
		encoder, ok := encoders[format]
		if !ok {
			log.Fatalf("unsupport format: %s\n", format)
		}

		if e := encoder(rd, &buffer); e != nil {
			log.Fatalf("failed to convert json to %s: %s\n", format, e)
		}
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Fatalln("failed to write output")
	}
}
