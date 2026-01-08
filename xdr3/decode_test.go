/*
 * Copyright (c) 2012-2014 Dave Collins <dave@davec.name>
 * Copyright (c) 2026 Stellar Development Foundation
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package xdr

import (
	"errors"
	"reflect"
	"testing"
)

// TestDecoder_EOF tests that Decoder returns proper errors on EOF
func TestDecoder_EOF(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		decode  func(d *Decoder) error
		errCode ErrorCode // expected error code, defaults to ErrIO
	}{
		{
			name: "DecodeInt EOF",
			data: []byte{0x00, 0x00}, // Only 2 bytes, need 4
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeInt()
				return err
			},
		},
		{
			name: "DecodeUint EOF",
			data: []byte{0x00, 0x00, 0x00}, // Only 3 bytes, need 4
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeUint()
				return err
			},
		},
		{
			name: "DecodeHyper EOF",
			data: []byte{0x00, 0x00, 0x00, 0x00}, // Only 4 bytes, need 8
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeHyper()
				return err
			},
		},
		{
			name: "DecodeUhyper EOF",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00}, // Only 5 bytes, need 8
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeUhyper()
				return err
			},
		},
		{
			name: "DecodeFloat EOF",
			data: []byte{0x00}, // Only 1 byte, need 4
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeFloat()
				return err
			},
		},
		{
			name: "DecodeDouble EOF",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // Only 6 bytes, need 8
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeDouble()
				return err
			},
		},
		{
			name: "DecodeBool EOF",
			data: []byte{}, // Empty
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeBool()
				return err
			},
		},
		{
			name: "DecodeFixedOpaque EOF",
			data: []byte{0x01, 0x02}, // Only 2 bytes, need 4 (size=3 padded)
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeFixedOpaque(3)
				return err
			},
		},
		{
			name: "DecodeFixedOpaqueInplace EOF",
			data: []byte{0x01, 0x02, 0x03}, // Only 3 bytes, need 4 (padded)
			decode: func(d *Decoder) error {
				out := make([]byte, 3)
				_, err := d.DecodeFixedOpaqueInplace(out)
				return err
			},
		},
		{
			name: "DecodeOpaque EOF on length",
			data: []byte{0x00, 0x00}, // Only 2 bytes for length prefix
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeOpaque(0)
				return err
			},
		},
		{
			name:    "DecodeOpaque EOF on data",
			data:    []byte{0x00, 0x00, 0x00, 0x08, 0x01, 0x02}, // Length=8, only 2 data bytes
			errCode: ErrOverflow,                                // length exceeds available data
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeOpaque(0)
				return err
			},
		},
		{
			name: "DecodeString EOF on length",
			data: []byte{0x00}, // Only 1 byte for length prefix
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeString(0)
				return err
			},
		},
		{
			name:    "DecodeString EOF on data",
			data:    []byte{0x00, 0x00, 0x00, 0x05, 'h', 'e'}, // Length=5, only 2 chars
			errCode: ErrOverflow,                              // length exceeds available data
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeString(0)
				return err
			},
		},
		{
			name: "Skip EOF",
			data: []byte{0x01, 0x02},
			decode: func(d *Decoder) error {
				return d.Skip(10)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.data)
			err := tt.decode(d)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Verify it's an UnmarshalError with expected error code
			var unmarshalErr *UnmarshalError
			if !errors.As(err, &unmarshalErr) {
				t.Fatalf("expected *UnmarshalError, got %T: %v", err, err)
			}
			expectedCode := tt.errCode
			if expectedCode == 0 {
				expectedCode = ErrIO
			}
			if unmarshalErr.ErrorCode != expectedCode {
				t.Errorf("expected %v, got %v", expectedCode, unmarshalErr.ErrorCode)
			}
		})
	}
}

// TestDecoder_InvalidPadding tests that non-zero padding bytes cause errors
func TestDecoder_InvalidPadding(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		decode func(d *Decoder) error
	}{
		{
			name: "DecodeFixedOpaque invalid padding",
			data: []byte{0x01, 0x02, 0x03, 0xFF}, // 3 bytes data + non-zero padding
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeFixedOpaque(3)
				return err
			},
		},
		{
			name: "DecodeFixedOpaqueInplace invalid padding",
			data: []byte{0x01, 0x02, 0x03, 0x01}, // 3 bytes data + non-zero padding
			decode: func(d *Decoder) error {
				out := make([]byte, 3)
				_, err := d.DecodeFixedOpaqueInplace(out)
				return err
			},
		},
		{
			name: "DecodeString invalid padding",
			data: []byte{
				0x00, 0x00, 0x00, 0x03, // length = 3
				'a', 'b', 'c', 0x01, // "abc" + non-zero padding
			},
			decode: func(d *Decoder) error {
				_, _, err := d.DecodeString(0)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.data)
			err := tt.decode(d)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unmarshalErr *UnmarshalError
			if !errors.As(err, &unmarshalErr) {
				t.Fatalf("expected *UnmarshalError, got %T: %v", err, err)
			}
			if unmarshalErr.ErrorCode != ErrIO {
				t.Errorf("expected ErrIO for padding error, got %v", unmarshalErr.ErrorCode)
			}
		})
	}
}

// TestDecoder_InvalidBool tests that invalid boolean values cause errors
func TestDecoder_InvalidBool(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"bool value 2", []byte{0x00, 0x00, 0x00, 0x02}},
		{"bool value -1", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"bool value 100", []byte{0x00, 0x00, 0x00, 0x64}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.data)
			_, _, err := d.DecodeBool()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unmarshalErr *UnmarshalError
			if !errors.As(err, &unmarshalErr) {
				t.Fatalf("expected *UnmarshalError, got %T: %v", err, err)
			}
			if unmarshalErr.ErrorCode != ErrBadEnumValue {
				t.Errorf("expected ErrBadEnumValue, got %v", unmarshalErr.ErrorCode)
			}
		})
	}
}

// TestDecoder_InvalidEnum tests that invalid enum values cause errors
func TestDecoder_InvalidEnum(t *testing.T) {
	validEnums := map[int32]bool{
		0: true,
		1: true,
		2: true,
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"enum value 3", []byte{0x00, 0x00, 0x00, 0x03}},
		{"enum value -1", []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"enum value 100", []byte{0x00, 0x00, 0x00, 0x64}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.data)
			_, _, err := d.DecodeEnum(validEnums)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unmarshalErr *UnmarshalError
			if !errors.As(err, &unmarshalErr) {
				t.Fatalf("expected *UnmarshalError, got %T: %v", err, err)
			}
			if unmarshalErr.ErrorCode != ErrBadEnumValue {
				t.Errorf("expected ErrBadEnumValue, got %v", unmarshalErr.ErrorCode)
			}
		})
	}
}

// TestDecoder_MaxSizeExceeded tests that exceeding max size causes errors
func TestDecoder_MaxSizeExceeded(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		maxSize int
		decode  func(d *Decoder, maxSize int) error
	}{
		{
			name:    "DecodeOpaque exceeds maxSize",
			data:    []byte{0x00, 0x00, 0x00, 0x10, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			maxSize: 8, // Only allow 8 bytes
			decode: func(d *Decoder, maxSize int) error {
				_, _, err := d.DecodeOpaque(maxSize)
				return err
			},
		},
		{
			name:    "DecodeString exceeds maxSize",
			data:    []byte{0x00, 0x00, 0x00, 0x08, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'},
			maxSize: 4, // Only allow 4 chars
			decode: func(d *Decoder, maxSize int) error {
				_, _, err := d.DecodeString(maxSize)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(tt.data)
			err := tt.decode(d, tt.maxSize)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unmarshalErr *UnmarshalError
			if !errors.As(err, &unmarshalErr) {
				t.Fatalf("expected *UnmarshalError, got %T: %v", err, err)
			}
			if unmarshalErr.ErrorCode != ErrOverflow {
				t.Errorf("expected ErrOverflow, got %v", unmarshalErr.ErrorCode)
			}
		})
	}
}

// TestDecoder_EmptyCases tests edge cases with empty/zero values
func TestDecoder_EmptyCases(t *testing.T) {
	t.Run("DecodeFixedOpaque size=0", func(t *testing.T) {
		d := NewDecoder([]byte{})
		data, n, err := d.DecodeFixedOpaque(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 0 {
			t.Errorf("expected empty slice, got %v", data)
		}
		if n != 0 {
			t.Errorf("expected 0 bytes read, got %d", n)
		}
	})

	t.Run("DecodeFixedOpaqueInplace size=0", func(t *testing.T) {
		d := NewDecoder([]byte{})
		n, err := d.DecodeFixedOpaqueInplace([]byte{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("expected 0 bytes read, got %d", n)
		}
	})

	t.Run("DecodeString empty", func(t *testing.T) {
		d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x00}) // length = 0
		s, n, err := d.DecodeString(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("expected empty string, got %q", s)
		}
		if n != 4 {
			t.Errorf("expected 4 bytes read (length prefix), got %d", n)
		}
	})
}

// TestDecoder_SuccessfulDecodes tests that valid data decodes correctly
func TestDecoder_SuccessfulDecodes(t *testing.T) {
	t.Run("DecodeInt", func(t *testing.T) {
		d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x2A}) // 42
		v, n, err := d.DecodeInt()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 42 {
			t.Errorf("expected 42, got %d", v)
		}
		if n != 4 {
			t.Errorf("expected 4 bytes, got %d", n)
		}
	})

	t.Run("DecodeInt negative", func(t *testing.T) {
		d := NewDecoder([]byte{0xFF, 0xFF, 0xFF, 0xFE}) // -2
		v, _, err := d.DecodeInt()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != -2 {
			t.Errorf("expected -2, got %d", v)
		}
	})

	t.Run("DecodeBool true", func(t *testing.T) {
		d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x01})
		v, _, err := d.DecodeBool()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !v {
			t.Error("expected true")
		}
	})

	t.Run("DecodeBool false", func(t *testing.T) {
		d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x00})
		v, _, err := d.DecodeBool()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v {
			t.Error("expected false")
		}
	})

	t.Run("DecodeEnum valid", func(t *testing.T) {
		validEnums := map[int32]bool{1: true, 2: true, 3: true}
		d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x02})
		v, _, err := d.DecodeEnum(validEnums)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 2 {
			t.Errorf("expected 2, got %d", v)
		}
	})

	t.Run("DecodeString with padding", func(t *testing.T) {
		// "abc" = 3 bytes, needs 1 byte padding
		d := NewDecoder([]byte{
			0x00, 0x00, 0x00, 0x03, // length = 3
			'a', 'b', 'c', 0x00, // data + padding
		})
		s, n, err := d.DecodeString(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != "abc" {
			t.Errorf("expected 'abc', got %q", s)
		}
		if n != 8 { // 4 (length) + 4 (padded data)
			t.Errorf("expected 8 bytes, got %d", n)
		}
	})

	t.Run("DecodeFixedOpaque with padding", func(t *testing.T) {
		// 5 bytes of data, needs 3 bytes padding
		d := NewDecoder([]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x00, 0x00, 0x00,
		})
		data, n, err := d.DecodeFixedOpaque(5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) != 5 {
			t.Errorf("expected 5 bytes, got %d", len(data))
		}
		if n != 8 { // 5 bytes padded to 8
			t.Errorf("expected 8 bytes read, got %d", n)
		}
	})
}

// TestDecoder_Position tests position tracking
func TestDecoder_Position(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0x01, // int32 = 1
		0x00, 0x00, 0x00, 0x02, // int32 = 2
	}
	d := NewDecoder(data)

	if d.Position() != 0 {
		t.Errorf("expected position 0, got %d", d.Position())
	}
	if d.Remaining() != 8 {
		t.Errorf("expected 8 remaining, got %d", d.Remaining())
	}

	d.DecodeInt()
	if d.Position() != 4 {
		t.Errorf("expected position 4, got %d", d.Position())
	}
	if d.Remaining() != 4 {
		t.Errorf("expected 4 remaining, got %d", d.Remaining())
	}

	d.DecodeInt()
	if d.Position() != 8 {
		t.Errorf("expected position 8, got %d", d.Position())
	}
	if d.Remaining() != 0 {
		t.Errorf("expected 0 remaining, got %d", d.Remaining())
	}
}

// TestDecoder_Reset tests the Reset method
func TestDecoder_Reset(t *testing.T) {
	d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x01})
	d.DecodeInt()

	if d.Position() != 4 {
		t.Errorf("expected position 4, got %d", d.Position())
	}

	// Reset with new data
	d.Reset([]byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03})
	if d.Position() != 0 {
		t.Errorf("expected position 0 after reset, got %d", d.Position())
	}
	if d.Remaining() != 8 {
		t.Errorf("expected 8 remaining after reset, got %d", d.Remaining())
	}

	v, _, _ := d.DecodeInt()
	if v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
}

// testDecoderFromType is a test type that implements DecoderFrom
type testDecoderFromType struct {
	Value int32
}

func (t *testDecoderFromType) DecodeFrom(d *Decoder, maxDepth uint) (int, error) {
	v, n, err := d.DecodeInt()
	if err != nil {
		return n, err
	}
	t.Value = v
	return n, nil
}

// testNestedType is a test type that tracks maxDepth to verify it's accessible
type testNestedType struct {
	ReceivedMaxDepth uint
	Value            int32
}

func (t *testNestedType) DecodeFrom(d *Decoder, maxDepth uint) (int, error) {
	t.ReceivedMaxDepth = d.MaxDepth()
	v, n, err := d.DecodeInt()
	if err != nil {
		return n, err
	}
	t.Value = v
	return n, nil
}

// TestUnmarshal tests the Unmarshal function with a type implementing DecoderFrom
func TestUnmarshal(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x2A} // 42 in big-endian

	var result testDecoderFromType
	n, err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes read, got %d", n)
	}
	if result.Value != 42 {
		t.Errorf("expected value 42, got %d", result.Value)
	}
}

// TestUnmarshal_ReflectionFallback tests Unmarshal with types that don't implement DecoderFrom
// but can be decoded via reflection
func TestUnmarshal_ReflectionFallback(t *testing.T) {
	// int32 doesn't implement DecoderFrom but can be decoded via reflection
	data := []byte{0x00, 0x00, 0x00, 0x01}
	var result int32
	n, err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("expected reflection decode to work, got error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes read, got %d", n)
	}
	if result != 1 {
		t.Errorf("expected result 1, got %d", result)
	}
}

// TestUnmarshal_ReflectionAllTypes tests reflection-based decoding for all primitive types
func TestUnmarshal_ReflectionAllTypes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		dest     interface{}
		expected interface{}
		wantN    int
	}{
		{
			name:     "int8",
			data:     []byte{0x00, 0x00, 0x00, 0x7F},
			dest:     new(int8),
			expected: int8(127),
			wantN:    4,
		},
		{
			name:     "int16",
			data:     []byte{0x00, 0x00, 0x7F, 0xFF},
			dest:     new(int16),
			expected: int16(32767),
			wantN:    4,
		},
		{
			name:     "int32",
			data:     []byte{0x7F, 0xFF, 0xFF, 0xFF},
			dest:     new(int32),
			expected: int32(2147483647),
			wantN:    4,
		},
		{
			name:     "int64",
			data:     []byte{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			dest:     new(int64),
			expected: int64(9223372036854775807),
			wantN:    8,
		},
		{
			name:     "uint8",
			data:     []byte{0x00, 0x00, 0x00, 0xFF},
			dest:     new(uint8),
			expected: uint8(255),
			wantN:    4,
		},
		{
			name:     "uint16",
			data:     []byte{0x00, 0x00, 0xFF, 0xFF},
			dest:     new(uint16),
			expected: uint16(65535),
			wantN:    4,
		},
		{
			name:     "uint32",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
			dest:     new(uint32),
			expected: uint32(4294967295),
			wantN:    4,
		},
		{
			name:     "uint64",
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			dest:     new(uint64),
			expected: uint64(18446744073709551615),
			wantN:    8,
		},
		{
			name:     "bool true",
			data:     []byte{0x00, 0x00, 0x00, 0x01},
			dest:     new(bool),
			expected: true,
			wantN:    4,
		},
		{
			name:     "bool false",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			dest:     new(bool),
			expected: false,
			wantN:    4,
		},
		{
			name:     "string",
			data:     []byte{0x00, 0x00, 0x00, 0x03, 'x', 'd', 'r', 0x00},
			dest:     new(string),
			expected: "xdr",
			wantN:    8,
		},
		{
			name:     "[]byte opaque",
			data:     []byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x02, 0x03, 0x00},
			dest:     new([]byte),
			expected: []byte{0x01, 0x02, 0x03},
			wantN:    8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := Unmarshal(tt.data, tt.dest)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if n != tt.wantN {
				t.Errorf("bytes read = %d, want %d", n, tt.wantN)
			}

			// Compare values using reflect
			got := reflect.ValueOf(tt.dest).Elem().Interface()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

// TestUnmarshal_ReflectionStruct tests reflection-based decoding for structs
func TestUnmarshal_ReflectionStruct(t *testing.T) {
	type simpleStruct struct {
		A int32
		B string
		C bool
	}

	// Encoded: A=42, B="hi", C=true
	data := []byte{
		0x00, 0x00, 0x00, 0x2A, // A = 42
		0x00, 0x00, 0x00, 0x02, 'h', 'i', 0x00, 0x00, // B = "hi" (padded)
		0x00, 0x00, 0x00, 0x01, // C = true
	}

	var result simpleStruct
	n, err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if n != 16 {
		t.Errorf("bytes read = %d, want 16", n)
	}
	if result.A != 42 {
		t.Errorf("A = %d, want 42", result.A)
	}
	if result.B != "hi" {
		t.Errorf("B = %q, want \"hi\"", result.B)
	}
	if result.C != true {
		t.Errorf("C = %v, want true", result.C)
	}
}

// TestUnmarshal_ReflectionSlice tests reflection-based decoding for slices
func TestUnmarshal_ReflectionSlice(t *testing.T) {
	// Slice of 3 int32s: [1, 2, 3]
	data := []byte{
		0x00, 0x00, 0x00, 0x03, // length = 3
		0x00, 0x00, 0x00, 0x01, // [0] = 1
		0x00, 0x00, 0x00, 0x02, // [1] = 2
		0x00, 0x00, 0x00, 0x03, // [2] = 3
	}

	var result []int32
	n, err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if n != 16 {
		t.Errorf("bytes read = %d, want 16", n)
	}
	expected := []int32{1, 2, 3}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

// TestUnmarshal_ReflectionMap tests reflection-based decoding for maps
func TestUnmarshal_ReflectionMap(t *testing.T) {
	// Map with 2 entries: {"a": 1, "b": 2}
	data := []byte{
		0x00, 0x00, 0x00, 0x02, // count = 2
		0x00, 0x00, 0x00, 0x01, 'a', 0x00, 0x00, 0x00, // key "a"
		0x00, 0x00, 0x00, 0x01, // value 1
		0x00, 0x00, 0x00, 0x01, 'b', 0x00, 0x00, 0x00, // key "b"
		0x00, 0x00, 0x00, 0x02, // value 2
	}

	var result map[string]int32
	n, err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if n != 28 {
		t.Errorf("bytes read = %d, want 28", n)
	}
	if result["a"] != 1 {
		t.Errorf("result[\"a\"] = %d, want 1", result["a"])
	}
	if result["b"] != 2 {
		t.Errorf("result[\"b\"] = %d, want 2", result["b"])
	}
}

// TestUnmarshal_ReflectionPointer tests reflection-based decoding for optional (pointer) types
func TestUnmarshal_ReflectionPointer(t *testing.T) {
	// Pointer present with value 42
	dataPresent := []byte{
		0x00, 0x00, 0x00, 0x01, // present = true
		0x00, 0x00, 0x00, 0x2A, // value = 42
	}

	var result1 *int32
	n, err := Unmarshal(dataPresent, &result1)
	if err != nil {
		t.Fatalf("Unmarshal (present) failed: %v", err)
	}
	if n != 8 {
		t.Errorf("bytes read = %d, want 8", n)
	}
	if result1 == nil || *result1 != 42 {
		t.Errorf("got %v, want pointer to 42", result1)
	}

	// Pointer absent (nil)
	dataAbsent := []byte{
		0x00, 0x00, 0x00, 0x00, // present = false
	}

	var result2 *int32
	result2 = new(int32) // Pre-allocate to verify it gets set to nil
	*result2 = 999
	n, err = Unmarshal(dataAbsent, &result2)
	if err != nil {
		t.Fatalf("Unmarshal (absent) failed: %v", err)
	}
	if n != 4 {
		t.Errorf("bytes read = %d, want 4", n)
	}
	if result2 != nil {
		t.Errorf("got %v, want nil", result2)
	}
}

// TestUnmarshalWithOptions tests UnmarshalWithOptions with custom MaxDepth
func TestUnmarshalWithOptions(t *testing.T) {
	data := []byte{0x00, 0x00, 0x01, 0x00} // 256 in big-endian

	var result testDecoderFromType
	opts := DecodeOptions{MaxDepth: 100}
	n, err := UnmarshalWithOptions(data, &result, opts)
	if err != nil {
		t.Fatalf("UnmarshalWithOptions failed: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes read, got %d", n)
	}
	if result.Value != 256 {
		t.Errorf("expected value 256, got %d", result.Value)
	}
}

// TestMaxDepthPassed tests that MaxDepth is correctly passed to DecodeFrom
func TestMaxDepthPassed(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x01}

	// Test with default options
	var result1 testNestedType
	_, err := Unmarshal(data, &result1)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if result1.ReceivedMaxDepth != DecodeDefaultMaxDepth {
		t.Errorf("expected maxDepth %d, got %d", DecodeDefaultMaxDepth, result1.ReceivedMaxDepth)
	}

	// Test with custom MaxDepth
	var result2 testNestedType
	opts := DecodeOptions{MaxDepth: 50}
	_, err = UnmarshalWithOptions(data, &result2, opts)
	if err != nil {
		t.Fatalf("UnmarshalWithOptions failed: %v", err)
	}
	if result2.ReceivedMaxDepth != 50 {
		t.Errorf("expected maxDepth 50, got %d", result2.ReceivedMaxDepth)
	}

	// Test with MaxDepth 0 (should use default)
	var result3 testNestedType
	opts = DecodeOptions{MaxDepth: 0}
	_, err = UnmarshalWithOptions(data, &result3, opts)
	if err != nil {
		t.Fatalf("UnmarshalWithOptions failed: %v", err)
	}
	if result3.ReceivedMaxDepth != DecodeDefaultMaxDepth {
		t.Errorf("expected maxDepth %d for zero option, got %d", DecodeDefaultMaxDepth, result3.ReceivedMaxDepth)
	}
}

// TestSkip_NegativeLength tests that negative skip length is rejected
func TestSkip_NegativeLength(t *testing.T) {
	d := NewDecoder([]byte{0x00, 0x00, 0x00, 0x00})
	err := d.Skip(-1)
	if err == nil {
		t.Fatal("expected error for negative skip length")
	}
	var unmarshalErr *UnmarshalError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("expected UnmarshalError, got %T", err)
	}
	if unmarshalErr.ErrorCode != ErrBadArguments {
		t.Errorf("expected ErrBadArguments, got %v", unmarshalErr.ErrorCode)
	}
}

// TestDecodeOpaque_LengthOverflow tests that length > maxInt32 is rejected
func TestDecodeOpaque_LengthOverflow(t *testing.T) {
	// Encode a length of 0x80000000 (2^31, which is > maxInt32)
	data := []byte{0x80, 0x00, 0x00, 0x00}
	d := NewDecoder(data)
	_, _, err := d.DecodeOpaque(0)
	if err == nil {
		t.Fatal("expected error for length > maxInt32")
	}
	var unmarshalErr *UnmarshalError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("expected UnmarshalError, got %T", err)
	}
	if unmarshalErr.ErrorCode != ErrOverflow {
		t.Errorf("expected ErrOverflow, got %v", unmarshalErr.ErrorCode)
	}
}

// TestDecodeString_LengthOverflow tests that length > maxInt32 is rejected
func TestDecodeString_LengthOverflow(t *testing.T) {
	// Encode a length of 0x80000000 (2^31, which is > maxInt32)
	data := []byte{0x80, 0x00, 0x00, 0x00}
	d := NewDecoder(data)
	_, _, err := d.DecodeString(0)
	if err == nil {
		t.Fatal("expected error for length > maxInt32")
	}
	var unmarshalErr *UnmarshalError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("expected UnmarshalError, got %T", err)
	}
	if unmarshalErr.ErrorCode != ErrOverflow {
		t.Errorf("expected ErrOverflow, got %v", unmarshalErr.ErrorCode)
	}
}

// TestDecoder_Decode tests the Decode convenience method
func TestDecoder_Decode(t *testing.T) {
	// Create decoder with initial data
	data1 := []byte{0x00, 0x00, 0x00, 0x2A} // 42
	decoder := NewDecoder(data1)

	var result testDecoderFromType
	n, err := decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes read, got %d", n)
	}
	if result.Value != 42 {
		t.Errorf("expected 42, got %d", result.Value)
	}

	// Test decoder reuse with Reset + Decode
	data2 := []byte{0x00, 0x00, 0x01, 0x00} // 256
	decoder.Reset(data2)

	n, err = decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Decode after Reset failed: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes read, got %d", n)
	}
	if result.Value != 256 {
		t.Errorf("expected 256, got %d", result.Value)
	}
}

// TestDecoder_Decode_MaxDepth tests that Decode passes MaxDepth correctly
func TestDecoder_Decode_MaxDepth(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x01}

	// Test with custom MaxDepth
	decoder := NewDecoderWithOptions(data, DecodeOptions{MaxDepth: 75})

	var result testNestedType
	_, err := decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if result.ReceivedMaxDepth != 75 {
		t.Errorf("expected maxDepth 75, got %d", result.ReceivedMaxDepth)
	}
}

// TestDecoder_Decode_Error tests that Decode propagates errors correctly
func TestDecoder_Decode_Error(t *testing.T) {
	// Insufficient data
	data := []byte{0x00, 0x00} // Only 2 bytes, need 4
	decoder := NewDecoder(data)

	var result testDecoderFromType
	_, err := decoder.Decode(&result)
	if err == nil {
		t.Fatal("expected error for insufficient data")
	}

	var unmarshalErr *UnmarshalError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("expected UnmarshalError, got %T", err)
	}
	if unmarshalErr.ErrorCode != ErrIO {
		t.Errorf("expected ErrIO, got %v", unmarshalErr.ErrorCode)
	}
}

// TestDecoder_MaxDepthExceeded tests that deeply nested structures trigger ErrMaxDecodingDepth
func TestDecoder_MaxDepthExceeded(t *testing.T) {
	// Create a linked list type with pointer to next
	type node struct {
		Value int32
		Next  *node
	}

	// Build XDR data for a linked list with 5 nodes
	// Each node: int32 value, bool (present), then next node
	// Node 1 -> Node 2 -> Node 3 -> Node 4 -> Node 5 -> nil
	data := []byte{
		// Node 1
		0x00, 0x00, 0x00, 0x01, // Value = 1
		0x00, 0x00, 0x00, 0x01, // Next present = true
		// Node 2
		0x00, 0x00, 0x00, 0x02, // Value = 2
		0x00, 0x00, 0x00, 0x01, // Next present = true
		// Node 3
		0x00, 0x00, 0x00, 0x03, // Value = 3
		0x00, 0x00, 0x00, 0x01, // Next present = true
		// Node 4
		0x00, 0x00, 0x00, 0x04, // Value = 4
		0x00, 0x00, 0x00, 0x01, // Next present = true
		// Node 5
		0x00, 0x00, 0x00, 0x05, // Value = 5
		0x00, 0x00, 0x00, 0x00, // Next present = false (nil)
	}

	// With default MaxDepth (200), this should succeed
	var result1 node
	_, err := Unmarshal(data, &result1)
	if err != nil {
		t.Fatalf("Unmarshal with default depth failed: %v", err)
	}

	// With MaxDepth = 3, this should fail (each pointer adds depth)
	// The structure needs: depth for node struct, depth for pointer, depth for next node...
	var result2 node
	opts := DecodeOptions{MaxDepth: 3}
	_, err = UnmarshalWithOptions(data, &result2, opts)
	if err == nil {
		t.Fatal("expected ErrMaxDecodingDepth error with MaxDepth=3")
	}

	var unmarshalErr *UnmarshalError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("expected UnmarshalError, got %T: %v", err, err)
	}
	if unmarshalErr.ErrorCode != ErrMaxDecodingDepth {
		t.Errorf("expected ErrMaxDecodingDepth, got %v", unmarshalErr.ErrorCode)
	}
}

// testUnionValueArm is a union type with value-type arms for testing
type testUnionValueArm struct {
	Type  int32
	Int   int32  // value type arm (switch 0)
	Str   string // value type arm (switch 1)
	Empty bool   // value type arm (switch 2)
}

func (u testUnionValueArm) ArmForSwitch(sw int32) (string, bool) {
	switch sw {
	case 0:
		return "Int", true
	case 1:
		return "Str", true
	case 2:
		return "", true // void arm
	}
	return "-", false
}

func (u testUnionValueArm) SwitchFieldName() string {
	return "Type"
}

// testUnionPointerArm is a union type with pointer-type arms for testing
type testUnionPointerArm struct {
	Type int32
	Int  *int32  // pointer type arm (switch 0)
	Str  *string // pointer type arm (switch 1)
}

func (u testUnionPointerArm) ArmForSwitch(sw int32) (string, bool) {
	switch sw {
	case 0:
		return "Int", true
	case 1:
		return "Str", true
	case 2:
		return "", true // void arm
	}
	return "-", false
}

func (u testUnionPointerArm) SwitchFieldName() string {
	return "Type"
}

// TestDecodeUnionWithValueTypeArm tests decoding unions with value-type arms
func TestDecodeUnionWithValueTypeArm(t *testing.T) {
	t.Run("value-type int arm", func(t *testing.T) {
		// XDR encoded union: Type=0, Int=42
		data := []byte{
			0x00, 0x00, 0x00, 0x00, // Type = 0
			0x00, 0x00, 0x00, 0x2A, // Int = 42
		}

		var result testUnionValueArm
		n, err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if n != 8 {
			t.Errorf("bytes read = %d, want 8", n)
		}
		if result.Type != 0 {
			t.Errorf("Type = %d, want 0", result.Type)
		}
		if result.Int != 42 {
			t.Errorf("Int = %d, want 42", result.Int)
		}
	})

	t.Run("value-type string arm", func(t *testing.T) {
		// XDR encoded union: Type=1, Str="hi"
		data := []byte{
			0x00, 0x00, 0x00, 0x01, // Type = 1
			0x00, 0x00, 0x00, 0x02, 'h', 'i', 0x00, 0x00, // Str = "hi" (padded)
		}

		var result testUnionValueArm
		n, err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if n != 12 {
			t.Errorf("bytes read = %d, want 12", n)
		}
		if result.Type != 1 {
			t.Errorf("Type = %d, want 1", result.Type)
		}
		if result.Str != "hi" {
			t.Errorf("Str = %q, want \"hi\"", result.Str)
		}
	})

	t.Run("void arm", func(t *testing.T) {
		// XDR encoded union: Type=2 (void arm, no data)
		data := []byte{
			0x00, 0x00, 0x00, 0x02, // Type = 2
		}

		var result testUnionValueArm
		n, err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if n != 4 {
			t.Errorf("bytes read = %d, want 4", n)
		}
		if result.Type != 2 {
			t.Errorf("Type = %d, want 2", result.Type)
		}
	})
}

// TestDecodeUnionWithPointerTypeArm tests decoding unions with pointer-type arms
func TestDecodeUnionWithPointerTypeArm(t *testing.T) {
	t.Run("pointer-type int arm", func(t *testing.T) {
		// XDR encoded union: Type=0, Int=42
		data := []byte{
			0x00, 0x00, 0x00, 0x00, // Type = 0
			0x00, 0x00, 0x00, 0x2A, // Int = 42
		}

		var result testUnionPointerArm
		n, err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if n != 8 {
			t.Errorf("bytes read = %d, want 8", n)
		}
		if result.Type != 0 {
			t.Errorf("Type = %d, want 0", result.Type)
		}
		if result.Int == nil || *result.Int != 42 {
			t.Errorf("Int = %v, want pointer to 42", result.Int)
		}
	})

	t.Run("pointer-type string arm", func(t *testing.T) {
		// XDR encoded union: Type=1, Str="hi"
		data := []byte{
			0x00, 0x00, 0x00, 0x01, // Type = 1
			0x00, 0x00, 0x00, 0x02, 'h', 'i', 0x00, 0x00, // Str = "hi" (padded)
		}

		var result testUnionPointerArm
		n, err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if n != 12 {
			t.Errorf("bytes read = %d, want 12", n)
		}
		if result.Type != 1 {
			t.Errorf("Type = %d, want 1", result.Type)
		}
		if result.Str == nil || *result.Str != "hi" {
			t.Errorf("Str = %v, want pointer to \"hi\"", result.Str)
		}
	})
}
