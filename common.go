package main

import (
	"fmt"
	"io"
	"reflect"
	"unicode/utf16"
	"unsafe"

	"github.com/xhebox/bstruct/byteorder"
)

var endian = byteorder.LittleEndian

type U8S []byte

func (s *U8S) Unmarshal(rd io.Reader) error {
	for i := 0; i < len(*s); {
		n, err := rd.Read((*s)[i:])
		if err != nil {
			return err
		}
		i += n
		//fmt.Printf("%d %d %d\n", i, len(*s), n)
	}
	return nil
}

func (s *U8S) Marshal(wt io.Writer) error {
	for i := 0; i < len(*s); {
		n, err := wt.Write((*s)[i:])
		if err != nil {
			return err
		}
		i += n
	}
	return nil
}

type U8 uint8

func (s *U8) Unmarshal(rd io.Reader) error {
	var buf [1]byte
	u8s := U8S(buf[:])
	err := u8s.Unmarshal(rd)
	*s = U8(buf[0])
	return err
}

func (s *U8) Marshal(wt io.Writer) error {
	var buf [1]byte
	buf[0] = uint8(*s)
	u8s := U8S(buf[:])
	return u8s.Marshal(wt)
}

type U16 uint16

func (s *U16) Unmarshal(rd io.Reader) error {
	var buf [2]byte
	u8s := U8S(buf[:])
	err := u8s.Unmarshal(rd)
	*s = U16(endian.Uint16(buf[:]))
	return err
}

func (s *U16) Marshal(wt io.Writer) error {
	var buf [2]byte
	endian.PutUint16(buf[:], uint16(*s))
	u8s := U8S(buf[:])
	return u8s.Marshal(wt)
}

type U32 uint32

func (s *U32) Unmarshal(rd io.Reader) error {
	var buf [4]byte
	u8s := U8S(buf[:])
	err := u8s.Unmarshal(rd)
	*s = U32(endian.Uint32(buf[:]))
	return err
}

func (s *U32) Marshal(wt io.Writer) error {
	var buf [4]byte
	endian.PutUint32(buf[:], uint32(*s))
	u8s := U8S(buf[:])
	return u8s.Marshal(wt)
}

type WideString string

func (s *WideString) Unmarshal(rd io.Reader) error {
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}

	buf := U8S(make([]byte, length*2))
	if err := buf.Unmarshal(rd); err != nil {
		return err
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&buf[0])),
		Len:  int(length),
		Cap:  int(length),
	}
	u16s := *(*[]uint16)(unsafe.Pointer(&hdr))
	*s = WideString(utf16.Decode(u16s))
	return nil
}

func (s *WideString) Marshal(wt io.Writer) error {
	u16s := utf16.Encode([]rune(*s))

	length := U32(len(u16s))
	if err := length.Marshal(wt); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}

	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&u16s[0])),
		Len:  int(length * 2),
		Cap:  int(length * 2),
	}
	u8s := U8S(*(*[]uint8)(unsafe.Pointer(&hdr)))
	return u8s.Marshal(wt)
}

func (s *WideString) Size() int {
	return 4 + len(utf16.Encode([]rune(*s)))*2
}

type WideStrings []string

func (s *WideStrings) Unmarshal(rd io.Reader) error {
	for {
		var length U32
		if err := length.Unmarshal(rd); err != nil {
			return fmt.Errorf("length -> %w", err)
		}

		if length == 0 {
			break
		}

		buf := U8S(make([]byte, length*2))
		if err := buf.Unmarshal(rd); err != nil {
			return fmt.Errorf("u16s -> %w", err)
		}
		hdr := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&buf[0])),
			Len:  int(length),
			Cap:  int(length),
		}
		u16s := *(*[]uint16)(unsafe.Pointer(&hdr))
		*s = append(*s, string(utf16.Decode(u16s)))
	}
	return nil
}

func (s *WideStrings) Marshal(wt io.Writer) error {
	for _, j := range *s {
		u16s := utf16.Encode([]rune(j))

		length := U32(len(u16s))
		if err := length.Marshal(wt); err != nil {
			return err
		}

		hdr := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&u16s[0])),
			Len:  int(length * 2),
			Cap:  int(length * 2),
		}
		u8s := U8S(*(*[]uint8)(unsafe.Pointer(&hdr)))
		if err := u8s.Marshal(wt); err != nil {
			return err
		}
	}
	terminator := U32(0)
	return terminator.Marshal(wt)
}

type Single struct {
	Un1    U32
	Un2    U32
	Tip   WideString
	Desc1 WideString
	Desc2 WideString
}

func (s *Single) Unmarshal(rd io.Reader) error {
	if err := s.Un1.Unmarshal(rd); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}
	if err := s.Un2.Unmarshal(rd); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}
	if err := s.Tip.Unmarshal(rd); err != nil {
		return fmt.Errorf("tip -> %w", err)
	}
	if err := s.Desc1.Unmarshal(rd); err != nil {
		return fmt.Errorf("desc1 -> %w", err)
	}
	if err := s.Desc2.Unmarshal(rd); err != nil {
		return fmt.Errorf("desc2 -> %w", err)
	}
	return nil
}

func (s *Single) Marshal(wt io.Writer) error {
	if err := s.Un1.Marshal(wt); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}
	if err := s.Un2.Marshal(wt); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}
	if err := s.Tip.Marshal(wt); err != nil {
		return fmt.Errorf("tip -> %w", err)
	}
	if err := s.Desc1.Marshal(wt); err != nil {
		return fmt.Errorf("desc1 -> %w", err)
	}
	if err := s.Desc2.Marshal(wt); err != nil {
		return fmt.Errorf("desc2 -> %w", err)
	}
	return nil
}
