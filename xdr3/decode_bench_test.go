/*
 * Copyright (c) 2012-2014 Dave Collins <dave@davec.name>
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

package xdr_test

import (
	"testing"

	xdr "github.com/stellar/go-xdr/xdr3"
)

// Test data for benchmarks
var (
	// XDR encoded int32 (value: 12345678)
	encodedInt = []byte{0x00, 0xBC, 0x61, 0x4E}

	// XDR encoded uint32 (value: 0xDEADBEEF)
	encodedUint = []byte{0xDE, 0xAD, 0xBE, 0xEF}

	// XDR encoded int64/hyper (value: 0x0102030405060708)
	encodedHyper = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	// XDR encoded string "hello world" (length 11, padded to 12 bytes)
	encodedString = []byte{
		0x00, 0x00, 0x00, 0x0B, // length = 11
		'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', 0x00, // "hello world" + 1 byte padding
	}

	// XDR encoded longer string (64 bytes of 'x', padded)
	encodedLongString = func() []byte {
		data := make([]byte, 4+64)                                  // length prefix + 64 chars (no padding needed, 64 % 4 == 0)
		data[0], data[1], data[2], data[3] = 0x00, 0x00, 0x00, 0x40 // length = 64
		for i := 4; i < 68; i++ {
			data[i] = 'x'
		}
		return data
	}()

	// XDR encoded variable opaque (32 bytes)
	encodedOpaque = func() []byte {
		data := make([]byte, 4+32)                                  // length prefix + 32 bytes
		data[0], data[1], data[2], data[3] = 0x00, 0x00, 0x00, 0x20 // length = 32
		for i := 4; i < 36; i++ {
			data[i] = byte(i - 4)
		}
		return data
	}()

	// XDR encoded fixed opaque (32 bytes, no length prefix)
	encodedFixedOpaque = func() []byte {
		data := make([]byte, 32)
		for i := 0; i < 32; i++ {
			data[i] = byte(i)
		}
		return data
	}()
)

// ============================================================================
// Primitive Decoding Benchmarks
// ============================================================================

func BenchmarkDecodeInt(b *testing.B) {
	d := xdr.NewDecoder(encodedInt)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedInt)
		_, _, _ = d.DecodeInt()
	}
	b.SetBytes(int64(len(encodedInt)))
}

func BenchmarkDecodeUint(b *testing.B) {
	d := xdr.NewDecoder(encodedUint)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedUint)
		_, _, _ = d.DecodeUint()
	}
	b.SetBytes(int64(len(encodedUint)))
}

func BenchmarkDecodeHyper(b *testing.B) {
	d := xdr.NewDecoder(encodedHyper)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedHyper)
		_, _, _ = d.DecodeHyper()
	}
	b.SetBytes(int64(len(encodedHyper)))
}

// ============================================================================
// String Decoding Benchmarks
// ============================================================================

func BenchmarkDecodeString(b *testing.B) {
	d := xdr.NewDecoder(encodedString)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedString)
		_, _, _ = d.DecodeString(0)
	}
	b.SetBytes(int64(len(encodedString)))
}

func BenchmarkDecodeLongString(b *testing.B) {
	d := xdr.NewDecoder(encodedLongString)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedLongString)
		_, _, _ = d.DecodeString(0)
	}
	b.SetBytes(int64(len(encodedLongString)))
}

// ============================================================================
// Opaque Decoding Benchmarks
// ============================================================================

func BenchmarkDecodeOpaque(b *testing.B) {
	d := xdr.NewDecoder(encodedOpaque)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedOpaque)
		_, _, _ = d.DecodeOpaque(0)
	}
	b.SetBytes(int64(len(encodedOpaque)))
}

// ============================================================================
// Fixed Opaque Decoding Benchmarks
// ============================================================================

func BenchmarkDecodeFixedOpaque(b *testing.B) {
	d := xdr.NewDecoder(encodedFixedOpaque)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedFixedOpaque)
		_, _, _ = d.DecodeFixedOpaque(32)
	}
	b.SetBytes(int64(len(encodedFixedOpaque)))
}

// ============================================================================
// Multiple Field Decoding Benchmarks (simulating struct decoding)
// ============================================================================

// Encoded struct with: uint32, int64, string "hello world", 32-byte opaque
var encodedStruct = func() []byte {
	var buf []byte
	buf = append(buf, encodedUint...)
	buf = append(buf, encodedHyper...)
	buf = append(buf, encodedString...)
	buf = append(buf, 0x00, 0x00, 0x00, 0x20) // opaque length = 32
	for i := 0; i < 32; i++ {
		buf = append(buf, byte(i))
	}
	return buf
}()

func BenchmarkDecodeMultipleFields(b *testing.B) {
	d := xdr.NewDecoder(encodedStruct)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(encodedStruct)
		_, _, _ = d.DecodeUint()
		_, _, _ = d.DecodeHyper()
		_, _, _ = d.DecodeString(0)
		_, _, _ = d.DecodeOpaque(0)
	}
	b.SetBytes(int64(len(encodedStruct)))
}

// ============================================================================
// Decoder Reuse Benchmarks
// ============================================================================

// benchDecoderFromType implements DecoderFrom for benchmarking
type benchDecoderFromType struct {
	Value int32
}

func (t *benchDecoderFromType) DecodeFrom(d *xdr.Decoder) (int, error) {
	v, n, err := d.DecodeInt()
	if err != nil {
		return n, err
	}
	t.Value = v
	return n, nil
}

// BenchmarkNewDecoderEachTime measures creating a new decoder for each decode
func BenchmarkNewDecoderEachTime(b *testing.B) {
	data := encodedInt
	var result benchDecoderFromType
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := xdr.NewDecoder(data)
		_, _ = d.Decode(&result)
	}
	b.SetBytes(int64(len(data)))
}

// BenchmarkDecoderReuse measures reusing a decoder with Reset+Decode
func BenchmarkDecoderReuse(b *testing.B) {
	data := encodedInt
	d := xdr.NewDecoder(nil)
	var result benchDecoderFromType
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Reset(data)
		_, _ = d.Decode(&result)
	}
	b.SetBytes(int64(len(data)))
}
