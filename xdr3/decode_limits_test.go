package xdr

import (
	"bytes"
	"encoding/binary"
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

// makeArrayPayload creates an XDR variable-length array payload of payloadSize
// bytes where the declared element count in the 4-byte header intentionally
// far exceeds the number of elements that actually fit in the remaining data.
// This simulates a malicious or malformed input that claims a huge array to
// provoke unbounded allocation — the decoder must rely on MaxOutputBytes to
// cap memory use before it encounters EOF while reading element data.
func makeArrayPayload(payloadSize int) []byte {
	payload := make([]byte, payloadSize)
	declaredLen := uint32(payloadSize - 4)
	binary.BigEndian.PutUint32(payload[0:4], declaredLen)
	return payload
}

func TestMaxOutputBytes(t *testing.T) {
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
			MaxOutputBytes: budget,
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
			MaxOutputBytes: budget,
		})
		if len(result) == 0 {
			t.Errorf("expected some decoded elements, err=%v", err)
		}
	})
}
