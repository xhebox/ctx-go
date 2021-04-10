package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Coder = func(rd io.Reader, wt io.Writer) error

var (
	encoders = map[string]Coder{
		"quests":    compileQuests,
		"ctx":       compileCtx,
		"glossary":  compileGlossary,
		"dialogues": compileDialogues,
	}
	decoders = map[string]Coder{
		"quests":    parseQuests,
		"ctx":       parseCtx,
		"glossary":  parseGlossary,
		"dialogues": parseDialogues,
	}
)

func getfmt(filename string) (string, string) {
	mode := "parse"
	if strings.HasSuffix(filename, ".json") {
		mode = "compile"
	}

	rfile := strings.TrimSuffix(filename, ".json")
	if strings.HasSuffix(rfile, ".ctx") {
		return "ctx", mode
	}
	rfile = path.Base(rfile)
	if strings.HasPrefix(rfile, "quests") {
		return "quests", mode
	} else if strings.HasPrefix(rfile, "glossary") {
		return "glossary", mode
	} else if strings.HasPrefix(rfile, "dialogues") {
		return "dialogues", mode
	}
	return filename, mode
}

func process(in, out, format string) error {
	var buffer bytes.Buffer

	in = path.Clean(in)
	out = path.Clean(out)

	realfmt, mode := getfmt(in)
	if format != "auto" {
		realfmt = format
	}

	switch mode {
	case "parse":
		decoder, ok := decoders[realfmt]
		if !ok {
			return fmt.Errorf("unsupport realfmt: %s\n", realfmt)
		}

		buf, e := ioutil.ReadFile(in)
		if e != nil {
			return fmt.Errorf("failed to read input: %w", e)
		}
		rd := bytes.NewReader(buf)

		if e := decoder(rd, &buffer); e != nil {
			return fmt.Errorf("failed to convert %s to json: %w\n", realfmt, e)
		}

		if !strings.HasSuffix(out, ".json") {
			out += ".json"
		}
	case "compile":
		encoder, ok := encoders[realfmt]
		if !ok {
			return fmt.Errorf("unsupport realfmt: %s\n", realfmt)
		}

		buf, e := ioutil.ReadFile(in)
		if e != nil {
			return fmt.Errorf("failed to read input: %w", e)
		}

		rd := bytes.NewReader(buf)
		if e := encoder(rd, &buffer); e != nil {
			return fmt.Errorf("failed to convert json to %s: %s\n", realfmt, e)
		}

		out = strings.TrimSuffix(out, ".json")
	}

	err := ioutil.WriteFile(out, buffer.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

func process_dir(in, out string) error {
	err := filepath.Walk(in, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() && p != in {
			return filepath.SkipDir
		}
		realfmt, _ := getfmt(p)
		if realfmt == p {
			return nil
		}
		log.Printf("%s\n", p)
		rel, err := filepath.Rel(in, p)
		if err != nil {
			return err
		}
		o := filepath.Join(out, rel)
		err = os.MkdirAll(filepath.Dir(o), 0755)
		if err != nil {
			return err
		}
		err = process(p, o, realfmt)
		if err != nil && strings.HasPrefix(err.Error(), "unsupport realfmt") {
			return nil
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("error walking the path %q: %v\n", in, err)
	}
	return nil
}

func main() {
	var in, out, format string
	flag.StringVar(&in, "i", "input", "input file [or directory].")
	flag.StringVar(&out, "o", "output", "output file [or directory].")
	flag.StringVar(&format, "f", "auto", "file format, one of ctx/, [ignored if input is a directory]")
	orig := flag.CommandLine.Usage
	flag.CommandLine.Usage = func() {
		orig()
		fmt.Printf("\n%s -i inputs.ctx, will unpack the file into 'output.json'. Formats are detected by file extension.\n", os.Args[0])
		fmt.Printf("%s -i output.json -o inputs.ctx.mod -f ctx, will pack the file into 'inputs.ctx.mod', with an explicit format hint. Packing or not is decided by if there is a '.json' suffix.\n", os.Args[0])
		fmt.Printf("\n%s -i path-to-spellforce -o unpack, will unpack all supported files in the game directory into dir 'unpack'. Unpacked files are suffixed with '.json'.\n", os.Args[0])
		fmt.Printf("%s -i unpack -o path-to-spellforce, will convert it back.\n", os.Args[0])
		fmt.Printf("\n%s -i dirs-of-files -o unpack, will work too. Program will detect bin_win32, bin_exp1_win32, content, text directories, and all supported file formats automatically.\n", os.Args[0])
	}
	flag.Parse()
	log.SetFlags(log.Llongfile)

	if i, err := os.Stat(in); err == nil && !i.IsDir() {
		err := process(in, out, format)
		if err != nil {
			log.Fatalf("fail to process the file %q: %v\n", in, err)
		}
	}

	err := filepath.Walk(in, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			var err error
			switch info.Name() {
			case "bin_win32":
				err = process_dir(filepath.Join(p, "content"), filepath.Join(out, "bin_win32", "content"))
				if err != nil {
					return err
				}
				err = process_dir(filepath.Join(p, "text"), filepath.Join(out, "bin_win32", "text"))
				if err != nil {
					return err
				}
			case "bin_exp1_win32":
				err = process_dir(filepath.Join(p, "content"), filepath.Join(out, "bin_exp1_win32", "content"))
				if err != nil {
					return err
				}
				err = process_dir(filepath.Join(p, "text"), filepath.Join(out, "bin_exp1_win32", "text"))
				if err != nil {
					return err
				}
			case "content", "text":
				err = process_dir(p, filepath.Join(out, info.Name()))
				if err != nil {
					return err
				}
			default:
				if p != in {
					return filepath.SkipDir
				} else {
					return process_dir(p, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking the path %q: %v\n", in, err)
	}
}
