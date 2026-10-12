// Package resources contains byte-level resource codecs, independent of client
// discovery and export.
package resources

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"

	"github.com/pierrec/lz4/v4"
	blitz "github.com/thecoooler/blitz-client"
)

const (
	footerSize       = 20
	maxPlaintextSize = 1024 * 1024 * 1024
)

var magic = []byte("DVPL")

// ContainerKind describes how a resource is stored: as plain bytes or inside a
// DVPL container with the given compression type.
type ContainerKind struct {
	// DVPL is false for plain resources.
	DVPL bool
	// Compression is the DVPL compression type: 0 none, 1/2 LZ4, 3 DEFLATE.
	Compression uint32
}

// Plain is the container kind of unwrapped resources.
var Plain = ContainerKind{}

// DVPL returns the container kind for a DVPL compression type.
func DVPL(compression uint32) ContainerKind {
	return ContainerKind{DVPL: true, Compression: compression}
}

func (k ContainerKind) String() string {
	if !k.DVPL {
		return "Plain"
	}
	return fmt.Sprintf("Dvpl(%d)", k.Compression)
}

// DecodeDVPL unwraps a DVPL container. Inputs without a DVPL footer are
// returned unchanged as Plain.
func DecodeDVPL(input []byte) ([]byte, ContainerKind, error) {
	if len(input) < footerSize || !bytes.Equal(input[len(input)-4:], magic) {
		return bytes.Clone(input), Plain, nil
	}
	footer := input[len(input)-footerSize:]
	plainSize := uint64(binary.LittleEndian.Uint32(footer[0:4]))
	compressedSize := uint64(binary.LittleEndian.Uint32(footer[4:8]))
	expectedCRC := binary.LittleEndian.Uint32(footer[8:12])
	compression := binary.LittleEndian.Uint32(footer[12:16])
	if plainSize > maxPlaintextSize {
		return nil, Plain, blitz.Errorf("DVPL_SIZE_LIMIT",
			"DVPL payload size %d exceeds the 1 GiB safety limit", plainSize)
	}
	found := uint64(len(input) - footerSize)
	if found != compressedSize {
		return nil, Plain, blitz.Errorf("DVPL_SIZE_MISMATCH",
			"DVPL footer declares %d payload bytes, found %d", compressedSize, found)
	}
	payload := input[:len(input)-footerSize]
	if actual := CRC32(payload); actual != expectedCRC {
		return nil, Plain, blitz.Errorf("DVPL_CRC_FAILED",
			"DVPL payload CRC mismatch: expected %08x, got %08x", expectedCRC, actual)
	}

	var decoded []byte
	switch compression {
	case 0:
		decoded = bytes.Clone(payload)
	case 1, 2:
		buffer := make([]byte, plainSize)
		n, err := lz4.UncompressBlock(payload, buffer)
		if err != nil && !(plainSize == 0 && len(payload) <= 1) {
			return nil, Plain, blitz.Errorf("DVPL_DECOMPRESS_FAILED", "LZ4 decompression failed: %v", err)
		}
		decoded = buffer[:n]
	case 3:
		reader := flate.NewReader(bytes.NewReader(payload))
		limited, err := io.ReadAll(io.LimitReader(reader, int64(plainSize)+1))
		if err == nil && uint64(len(limited)) > plainSize {
			err = fmt.Errorf("output exceeds the declared size %d", plainSize)
		}
		if err != nil {
			return nil, Plain, blitz.Errorf("DVPL_DECOMPRESS_FAILED", "DEFLATE decompression failed: %v", err)
		}
		decoded = limited
	default:
		return nil, Plain, blitz.Errorf("DVPL_COMPRESSION_UNSUPPORTED",
			"unsupported DVPL compression type %d", compression)
	}
	if uint64(len(decoded)) != plainSize {
		return nil, Plain, blitz.Errorf("DVPL_SIZE_MISMATCH",
			"DVPL declares %d plaintext bytes, decoded %d", plainSize, len(decoded))
	}
	return decoded, DVPL(compression), nil
}

// EncodeDVPL wraps plain bytes in the given container kind.
func EncodeDVPL(plain []byte, kind ContainerKind) ([]byte, error) {
	if !kind.DVPL {
		return bytes.Clone(plain), nil
	}
	if len(plain) > maxPlaintextSize || uint64(len(plain)) > math.MaxUint32 {
		return nil, blitz.NewError("DVPL_SIZE_LIMIT", "patched resource is too large for a DVPL container")
	}
	var payload []byte
	switch kind.Compression {
	case 0:
		payload = bytes.Clone(plain)
	case 1, 2:
		payload = compressLZ4(plain)
	case 3:
		var buffer bytes.Buffer
		writer, _ := flate.NewWriter(&buffer, 6)
		writer.Write(plain)
		writer.Close()
		payload = buffer.Bytes()
	default:
		return nil, blitz.Errorf("DVPL_COMPRESSION_UNSUPPORTED",
			"unsupported DVPL compression type %d", kind.Compression)
	}
	if uint64(len(payload)) > math.MaxUint32 {
		return nil, blitz.NewError("DVPL_SIZE_LIMIT", "compressed DVPL payload is too large")
	}

	output := make([]byte, 0, len(payload)+footerSize)
	output = append(output, payload...)
	output = binary.LittleEndian.AppendUint32(output, uint32(len(plain)))
	output = binary.LittleEndian.AppendUint32(output, uint32(len(payload)))
	output = binary.LittleEndian.AppendUint32(output, CRC32(payload))
	output = binary.LittleEndian.AppendUint32(output, kind.Compression)
	output = append(output, magic...)
	return output, nil
}

func compressLZ4(plain []byte) []byte {
	if len(plain) == 0 {
		// A single token with no literals and no match.
		return []byte{0}
	}
	buffer := make([]byte, lz4.CompressBlockBound(len(plain)))
	var compressor lz4.Compressor
	n, err := compressor.CompressBlock(plain, buffer)
	if err != nil || n == 0 {
		return literalLZ4Block(plain)
	}
	return buffer[:n]
}

// literalLZ4Block encodes data as one literal-only LZ4 sequence.
func literalLZ4Block(plain []byte) []byte {
	output := make([]byte, 0, len(plain)+len(plain)/255+16)
	length := len(plain)
	if length < 15 {
		output = append(output, byte(length<<4))
	} else {
		output = append(output, 0xf0)
		rest := length - 15
		for rest >= 255 {
			output = append(output, 255)
			rest -= 255
		}
		output = append(output, byte(rest))
	}
	return append(output, plain...)
}

// CRC32 returns the IEEE CRC-32 checksum used by DVPL footers.
func CRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}
