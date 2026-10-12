package resources

import (
	"bytes"
	"testing"

	blitz "github.com/thecoooler/blitz-client"
)

func TestCodecsAreReusableWithoutClientDiscovery(t *testing.T) {
	plain := []byte("resource payload, resource payload, resource payload")
	for _, kind := range []ContainerKind{Plain, DVPL(0), DVPL(1), DVPL(2), DVPL(3)} {
		encoded, err := EncodeDVPL(plain, kind)
		if err != nil {
			t.Fatal(err)
		}
		decoded, detected, err := DecodeDVPL(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded, plain) || detected != kind {
			t.Fatalf("%v: got %q %v", kind, decoded, detected)
		}
		if kind != Plain {
			damaged := bytes.Clone(encoded)
			damaged[0] ^= 1
			_, _, err := DecodeDVPL(damaged)
			if blitz.AsError(err).Code != "DVPL_CRC_FAILED" {
				t.Fatalf("%v: %v", kind, err)
			}
		}
	}
}

func TestEmptyAndIncompressiblePayloadsRoundTrip(t *testing.T) {
	random := make([]byte, 4096)
	state := uint32(1)
	for index := range random {
		state = state*1664525 + 1013904223
		random[index] = byte(state >> 24)
	}
	for _, plain := range [][]byte{{}, {1}, random} {
		for _, kind := range []ContainerKind{DVPL(0), DVPL(1), DVPL(2), DVPL(3)} {
			encoded, err := EncodeDVPL(plain, kind)
			if err != nil {
				t.Fatal(err)
			}
			decoded, detected, err := DecodeDVPL(encoded)
			if err != nil {
				t.Fatalf("%v (%d bytes): %v", kind, len(plain), err)
			}
			if !bytes.Equal(decoded, plain) || detected != kind {
				t.Fatalf("%v: round trip changed %d bytes", kind, len(plain))
			}
		}
	}
	literal := literalLZ4Block(random)
	decoded, _, err := DecodeDVPL(append(literal, footer(len(random), literal, 1)...))
	if err != nil || !bytes.Equal(decoded, random) {
		t.Fatalf("literal block: %v", err)
	}
}

func footer(plain int, payload []byte, compression uint32) []byte {
	out := []byte{}
	for _, value := range []uint32{uint32(plain), uint32(len(payload)), CRC32(payload), compression} {
		out = append(out, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
	}
	return append(out, "DVPL"...)
}

func TestRejectsInvalidContainers(t *testing.T) {
	encoded, _ := EncodeDVPL([]byte("payload"), DVPL(0))
	truncated := append([]byte{}, encoded[1:]...)
	if _, _, err := DecodeDVPL(truncated); blitz.AsError(err).Code != "DVPL_SIZE_MISMATCH" {
		t.Fatal(err)
	}
	unsupported := append([]byte("x"), footer(1, []byte("x"), 9)...)
	if _, _, err := DecodeDVPL(unsupported); blitz.AsError(err).Code != "DVPL_COMPRESSION_UNSUPPORTED" {
		t.Fatal(err)
	}
	if _, err := EncodeDVPL(nil, DVPL(9)); blitz.AsError(err).Code != "DVPL_COMPRESSION_UNSUPPORTED" {
		t.Fatal(err)
	}
}
