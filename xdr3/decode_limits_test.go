package xdr

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"unsafe"
)

// wideStruct has a large in-memory footprint (256 bytes) relative to its
// minimal XDR wire representation.
type wideStruct struct {
	F0, F1, F2, F3, F4, F5, F6, F7         int64
	F8, F9, F10, F11, F12, F13, F14, F15   int64
	F16, F17, F18, F19, F20, F21, F22, F23 int64
	F24, F25, F26, F27, F28, F29, F30, F31 int64
}

// makeArrayPayload creates an XDR-encoded variable-length array header
// followed by zero-filled element data.
func makeArrayPayload(payloadSize int) []byte {
	payload := make([]byte, payloadSize)
	declaredLen := uint32(payloadSize - 4)
	binary.BigEndian.PutUint32(payload[0:4], declaredLen)
	return payload
}

func TestMaxMemoryBytes(t *testing.T) {
	payloadSize := 100000
	payload := makeArrayPayload(payloadSize)
	structSize := int64(unsafe.Sizeof(wideStruct{})) // 256

	t.Run("unlimited", func(t *testing.T) {
		var result []wideStruct
		reader := bytes.NewReader(payload)
		_, err := UnmarshalWithOptions(reader, &result, DecodeOptions{})
		if len(result) == 0 {
			t.Errorf("expected some decoded elements with no limit, err=%v", err)
		}
	})

	t.Run("exceeded", func(t *testing.T) {
		budget := int64(256) * structSize // 65536 bytes
		var result []wideStruct
		reader := bytes.NewReader(payload)
		_, err := UnmarshalWithOptions(reader, &result, DecodeOptions{
			MaxMemoryBytes: budget,
		})
		if err == nil {
			t.Error("expected error when output byte limit exceeded")
		}
		maxExpected := int(budget/structSize) + 1
		if len(result) > maxExpected {
			t.Errorf("decoded %d elements, expected at most %d", len(result), maxExpected)
		}
	})

	t.Run("not_reached", func(t *testing.T) {
		// Budget larger than what the payload can produce — decode
		// should succeed without hitting the limit.
		budget := int64(3000) * structSize
		var result []wideStruct
		reader := bytes.NewReader(payload)
		_, err := UnmarshalWithOptions(reader, &result, DecodeOptions{
			MaxMemoryBytes: budget,
		})
		if errors.Is(err, ErrMemoryLimitExceeded) {
			t.Errorf("expected budget not to be exceeded, got %v", err)
		}
		if len(result) == 0 {
			t.Errorf("expected some decoded elements before hitting end of input, err=%v", err)
		}
	})
}

// passthroughReader is an io.Reader that does not implement lenLeft, so
// NewDecoderWithOptions cannot tighten the MaxInputLen budget from the inner
// reader's remaining length.
type passthroughReader struct {
	r *bytes.Reader
}

func (p *passthroughReader) Read(b []byte) (int, error) { return p.r.Read(b) }

// TestMaxInputLenEnforced verifies that readerLenWrapper refuses reads past
// the configured MaxInputLen budget even when the underlying reader still has
// bytes available.
func TestMaxInputLenEnforced(t *testing.T) {
	// 4 bytes to fill the budget plus 4 extras the wrapper must refuse.
	payload := []byte{
		0x00, 0x00, 0x00, 0x42,
		0xff, 0xff, 0xff, 0xff,
	}
	reader := &passthroughReader{r: bytes.NewReader(payload)}

	dec := NewDecoderWithOptions(reader, DecodeOptions{MaxInputLen: 4})

	if _, _, err := dec.DecodeInt(); err != nil {
		t.Fatalf("DecodeInt: %v", err)
	}

	// Inner reader still holds 4 bytes; the wrapper must refuse them.
	buf := make([]byte, 4)
	n, err := dec.r.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("wrapper Read past budget: got (%d, %v), want (0, io.EOF)", n, err)
	}

	if left, ok := dec.InputLen(); !ok || left != 0 {
		t.Errorf("after budget exhaustion: InputLen=(%d,%v), want (0,true)", left, ok)
	}
}

// TestMaxInputLenDecodeStringBypass reproduces the reporter PoC end-to-end:
// a length prefix decoded past the MaxInputLen boundary must not reach the
// DecodeFixedOpaque allocation phase.
func TestMaxInputLenDecodeStringBypass(t *testing.T) {
	payload := []byte{
		0x00, 0x00, 0x00, 0x42,
		0x7f, 0xff, 0xff, 0xfb,
	}
	reader := &passthroughReader{r: bytes.NewReader(payload)}
	dec := NewDecoderWithOptions(reader, DecodeOptions{MaxInputLen: 4})

	if _, _, err := dec.DecodeInt(); err != nil {
		t.Fatalf("DecodeInt: %v", err)
	}

	if _, _, err := dec.DecodeString(0); err == nil {
		t.Fatal("DecodeString: expected error, got nil")
	}

	// Wrapper must refuse the length-prefix read, so the 4 bytes are still
	// in the inner reader. If the wrapper leaked past budget, the decoder
	// would have consumed them and proceeded to the allocation phase.
	if got := reader.r.Len(); got != 4 {
		t.Errorf("inner reader: got %d bytes remaining, want 4", got)
	}
}

// TestMaxInputLenPrefixAtEndOfInput is the regression for the length-prefix-at-
// end-of-input bypass: when a variable-length field's 4-byte length prefix is
// the final 4 bytes of the input, InputLen() reports 0 bytes remaining. That 0
// is a genuine bound (reject any non-zero declared length) and must not be
// mistaken for "no limit configured". The rejection must therefore come from the
// DecodeString bound check, before DecodeFixedOpaque runs make([]byte, dataLen).
func TestMaxInputLenPrefixAtEndOfInput(t *testing.T) {
	cases := []struct {
		name     string
		declared uint32
	}{
		{"huge declared length (2 GiB allocation averted)", 0x7fffffff},
		{"small declared length still rejected", 1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// [int][string length prefix]; the prefix is the last 4 bytes.
			payload := make([]byte, 8)
			binary.BigEndian.PutUint32(payload[0:4], 0x42)
			binary.BigEndian.PutUint32(payload[4:8], tc.declared)

			// MaxInputLen spans the whole input, so no bytes remain once the
			// prefix is read — the shape SafeUnmarshalBase64 produces for an
			// exact buffer.
			dec := NewDecoderWithOptions(bytes.NewReader(payload),
				DecodeOptions{MaxInputLen: len(payload)})

			if _, _, err := dec.DecodeInt(); err != nil {
				t.Fatalf("DecodeInt: %v", err)
			}

			_, _, err := dec.DecodeString(0)
			if err == nil {
				t.Fatal("DecodeString: expected rejection, got nil")
			}

			var ue *UnmarshalError
			if !errors.As(err, &ue) {
				t.Fatalf("DecodeString: error %v is not an *UnmarshalError", err)
			}
			// Func must be DecodeString (the bound check), not
			// DecodeFixedOpaqueInplace (which only fails after the allocation).
			if ue.Func != "DecodeString" || ue.ErrorCode != ErrOverflow {
				t.Errorf("rejected by xdr:%s (%v); want DecodeString/ErrOverflow — "+
					"bound deleted and allocation reached", ue.Func, ue.ErrorCode)
			}
		})
	}
}

// TestMaxInputLenSlicePrefixAtEndOfInput covers the same bypass through the
// non-opaque slice path (decodeArray). A []uint32 whose length prefix is the
// whole input declares a large element count with no element data following;
// the fix must reject at the decodeArray bound check, before reflect.MakeSlice
// and the element-decode loop are reached.
func TestMaxInputLenSlicePrefixAtEndOfInput(t *testing.T) {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload[0:4], 0x7fffffff)

	var out []uint32
	_, err := UnmarshalWithOptions(bytes.NewReader(payload), &out,
		DecodeOptions{MaxInputLen: len(payload)})
	if err == nil {
		t.Fatal("expected rejection, got nil")
	}

	var ue *UnmarshalError
	if !errors.As(err, &ue) {
		t.Fatalf("error %v is not an *UnmarshalError", err)
	}
	if ue.Func != "decodeArray" || ue.ErrorCode != ErrOverflow {
		t.Errorf("rejected by xdr:%s (%v); want decodeArray/ErrOverflow — "+
			"bound deleted and allocation reached", ue.Func, ue.ErrorCode)
	}
}

// TestMaxInputLenBudget verifies that the wrapper's budget is the minimum of
// MaxInputLen and the inner reader's reported remaining length, regardless of
// which is tighter. InputLen() reports the actual initial budget.
func TestMaxInputLenBudget(t *testing.T) {
	cases := []struct {
		name        string
		payloadLen  int
		maxInputLen int
		wantBudget  int
	}{
		{"MaxInputLen tighter than inner", 16, 4, 4},
		{"inner tighter than MaxInputLen", 4, 1 << 20, 4},
		{"equal", 8, 8, 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader(make([]byte, tc.payloadLen))
			dec := NewDecoderWithOptions(reader, DecodeOptions{MaxInputLen: tc.maxInputLen})
			if left, ok := dec.InputLen(); !ok || left != tc.wantBudget {
				t.Errorf("InputLen=(%d,%v), want (%d,true)", left, ok, tc.wantBudget)
			}
		})
	}
}

// negLenReader implements lenLeft but reports a negative remaining length,
// decoupled from the bytes it actually holds. With MaxInputLen unset the
// decoder trusts this Len() directly (d.l = r), so it drives InputLen()
// negative — the input shape decodeMap's clamp must tolerate.
type negLenReader struct{ r *bytes.Reader }

func (n *negLenReader) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n *negLenReader) Len() int                   { return -1 }

// TestDecodeMapNegativeInputLenClamped verifies decodeMap's guard clamps a
// negative InputLen to 0 rather than letting uint(negative) wrap to a huge
// value and bypass the check. The map length prefix declares one entry with no
// entry data following; the guard must reject before the decode loop runs.
func TestDecodeMapNegativeInputLenClamped(t *testing.T) {
	payload := []byte{0x00, 0x00, 0x00, 0x01} // map length = 1, then nothing

	var m map[uint32]uint32
	_, err := UnmarshalWithOptions(&negLenReader{r: bytes.NewReader(payload)}, &m,
		DecodeOptions{})
	if err == nil {
		t.Fatal("expected rejection, got nil")
	}

	var ue *UnmarshalError
	if !errors.As(err, &ue) {
		t.Fatalf("error %v is not an *UnmarshalError", err)
	}
	if ue.Func != "decodeMap" || ue.ErrorCode != ErrOverflow {
		t.Errorf("rejected by xdr:%s (%v); want decodeMap/ErrOverflow — "+
			"negative InputLen wrapped and bypassed the guard", ue.Func, ue.ErrorCode)
	}
}
