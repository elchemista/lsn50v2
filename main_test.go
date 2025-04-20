package main

import (
	"encoding/base64"
	"reflect"
	"testing"
)

// helper to encode raw bytes to base64
func encode(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func TestDecodeInvalidBase64(t *testing.T) {
	d := NewDecoder()
	_, err := d.Decode("not-base64!!")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestDecodePayloadTooShort(t *testing.T) {
	d := NewDecoder()
	// raw length < 7
	short := make([]byte, 5)
	b64 := encode(short)
	_, err := d.Decode(b64)
	if err == nil || err.Error() != "payload too short: 5 bytes" {
		t.Fatalf("expected payload too short error, got %v", err)
	}
}

func TestDecodeUnsupportedMode(t *testing.T) {
	raw := make([]byte, 8)
	// set mode bits to 6: raw[6] bits 2-6 = 6<<2
	raw[6] = byte(6 << 2)
	b64 := encode(raw)
	d := NewDecoder()
	_, err := d.Decode(b64)
	if err == nil || !contains(err.Error(), "unsupported mode 6") {
		t.Fatalf("expected unsupported mode 6 error, got %v", err)
	}
}

func TestDecodeMode1(t *testing.T) {
	// Construct raw payload for mode 1
	raw := make([]byte, 11)
	// band code at raw[0]
	raw[0] = 0x01 // EU868
	// set battery: bytes 0-1: 0x00,0x64 => 100 => 100/1000=0.1
	raw[0], raw[1] = 0x00, 0x64
	// temp1 invalid marker: r2-3 = 0x7f,0xff
	raw[2], raw[3] = 0x7f, 0xff
	// adc0 invalid for mode1? mode !=8
	raw[4], raw[5] = 0x00, 0xC8 // 200/1000=0.2
	// mode bits: raw[6] bits = mode1 (1<<2)
	raw[6] = byte(1 << 2)
	// distance bytes 7-8: 0x01,0x2C => 300 => /10=30.0cm
	raw[7], raw[8] = 0x01, 0x2C
	// signal bytes 9-10: 0x00,0x64 => 100
	raw[9], raw[10] = 0x00, 0x64

	b64 := encode(raw)
	d := NewDecoder()
	m, err := d.Decode(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// expect 3 measurements: Bat V, ADC CH0V, Distance Cm, Signal => commonMetrics + two
	expected := []Measurement{
		{"Bat V", 0.1},
		{"ADC CH0V", 0.2},
		{"Distance Cm", 30.0},
		{"Signal", 100.0},
	}
	if !reflect.DeepEqual(m, expected) {
		t.Errorf("mode1 decode mismatch. got %v, want %v", m, expected)
	}
}

// helper to check substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && reflect.ValueOf(s).String() != "" && (func() bool {
		return len(substr) == 0 || (len(s) >= len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
	})()
}
