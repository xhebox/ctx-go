package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
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

	switch format {
	case "ctx":
		if e := handleCtx(mode, rd, &buffer); e != nil {
			log.Fatalln(e);
		}
	case "quests":
		if e := handleQuests(mode, rd, &buffer); e != nil {
			log.Fatalln(e);
		}
	default:
		log.Fatalf("unsupport format: %s\n", format)
	}

	e = ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if e != nil {
		log.Fatalln("failed to write output")
	}
}
