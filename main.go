package main

import (
	"encoding/base64"
	"fmt"
)

// Decoder dispatches payloads to mode handlers.
type Decoder struct {
	handlers map[int]ModeHandler
}

// Measurement holds a named metric value.
type Measurement struct {
	Name  string
	Value float64
}

// ModeHandler decodes a Packet for its work mode.
type ModeHandler interface {
	Decode(*Packet) ([]Measurement, error)
}

// Packet holds raw payload and header fields.
type Packet struct {
	Raw  []byte
	Mode int
	Band string
}

// NewDecoder sets up handlers for modes 0–8 (mode 6 unsupported).
func NewDecoder() *Decoder {
	d := &Decoder{handlers: make(map[int]ModeHandler)}
	d.handlers[0] = mode0{}
	d.handlers[1] = mode1{}
	d.handlers[2] = mode2{}
	d.handlers[3] = mode3{}
	d.handlers[4] = mode4{}
	d.handlers[5] = mode5{}
	d.handlers[7] = mode7{}
	d.handlers[8] = mode8{}
	return d
}

// Decode base64 payload into measurements.
func (d *Decoder) Decode(b64 string) ([]Measurement, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}
	if len(raw) < 7 {
		return nil, fmt.Errorf("payload too short: %d bytes", len(raw))
	}
	p := &Packet{
		Raw:  raw,
		Mode: int((raw[6] & 0x7C) >> 2), // extract work mode bits
		Band: getBand(raw[0]),
	}
	handler, ok := d.handlers[p.Mode]
	if !ok {
		return nil, fmt.Errorf("unsupported mode %d", p.Mode)
	}
	return handler.Decode(p)
}

func getBand(code byte) string {
	switch code {
	case 0x01:
		return "EU868"
	case 0x02:
		return "US915"
	case 0x03:
		return "IN865"
	case 0x04:
		return "AU915"
	case 0x05:
		return "KZ865"
	case 0x06:
		return "RU864"
	case 0x07:
		return "AS923"
	case 0x08:
		return "AS923_1"
	case 0x09:
		return "AS923_2"
	case 0x0A:
		return "AS923_3"
	case 0x0B:
		return "CN470"
	case 0x0C:
		return "EU433"
	case 0x0D:
		return "KR920"
	case 0x0E:
		return "MA869"
	case 0x0F:
		return "AS923_4"
	default:
		return ""
	}
}

// commonMetrics handles battery, Temp C1, and ADC CH0 for most modes.
func commonMetrics(p *Packet) []Measurement {
	r := p.Raw
	metrics := make([]Measurement, 0)
	if p.Mode != 2 {
		bat := float64(int64(r[0])<<8|int64(r[1])) / 1000
		metrics = append(metrics, Measurement{"Bat V", bat})
		if !(r[2] == 0x7f && r[3] == 0xff) {
			signed := int32(r[2])<<24>>16 | int32(r[3])
			t := float64(signed) / 10
			metrics = append(metrics, Measurement{"Temp C1", t})
		}
		if p.Mode != 8 {
			adc0 := float64(int64(r[4])<<8|int64(r[5])) / 1000
			metrics = append(metrics, Measurement{"ADC CH0V", adc0})
		}
	}
	return metrics
}

// mode0: 3ADC, SHT, Illum.
type mode0 struct{}

func (mode0) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := commonMetrics(p)
	if (int(r[9])<<8 | int(r[10])) == 0 {
		illum := float64(int64(r[7])<<8 | int64(r[8]))
		m = append(m, Measurement{"Illum", illum})
	}
	if !((r[7] == 0x7f && r[8] == 0xff) || (r[7] == 0xff && r[8] == 0xff)) {
		signed := int32(r[7])<<24>>16 | int32(r[8])
		t := float64(signed) / 10
		m = append(m, Measurement{"TempC SHT", t})
		if !(r[9] == 0xff && r[10] == 0xff) {
			h := float64(int32(r[9])<<8|int32(r[10])) / 10
			m = append(m, Measurement{"Hum SHT", h})
		}
	}
	return m, nil
}

// mode1: distance, signal.
type mode1 struct{}

func (mode1) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := commonMetrics(p)
	if !(r[7] == 0x00 && r[8] == 0x00) {
		d := float64(int64(r[7])<<8|int64(r[8])) / 10
		m = append(m, Measurement{"Distance Cm", d})
	}
	if !(r[9] == 0xff && r[10] == 0xff) {
		s := float64(int64(r[9])<<8 | int64(r[10]))
		m = append(m, Measurement{"Signal", s})
	}
	return m, nil
}

// mode2: 3ADC+IIC.
type mode2 struct{}

func (mode2) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := []Measurement{{"Bat V", float64(r[11]) / 10}}
	m = append(m,
		Measurement{"ADC CH0V", float64(int64(r[0])<<8|int64(r[1])) / 1000},
		Measurement{"ADC CH1V", float64(int64(r[2])<<8|int64(r[3])) / 1000},
		Measurement{"ADC CH4V", float64(int64(r[4])<<8|int64(r[5])) / 1000},
	)
	if (int(r[9])<<8 | int(r[10])) == 0 {
		m = append(m, Measurement{"Illum", float64(int64(r[7])<<8 | int64(r[8]))})
	} else {
		if !((r[7] == 0x7f && r[8] == 0xff) || (r[7] == 0xff && r[8] == 0xff)) {
			t := float64(int32(r[7])<<24>>16|int32(r[8])) / 10
			m = append(m, Measurement{"TempC SHT", t})
		}
		if !(r[9] == 0xff && r[10] == 0xff) {
			h := float64(int32(r[9])<<8|int32(r[10])) / 10
			m = append(m, Measurement{"Hum SHT", h})
		}
	}
	return m, nil
}

// mode3: two DS18B20 temps.
type mode3 struct{}

func (mode3) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := commonMetrics(p)
	if !(r[7] == 0x7f && r[8] == 0xff) {
		t := float64(int32(r[7])<<24>>16|int32(r[8])) / 10
		m = append(m, Measurement{"Temp C2", t})
	}
	if !(r[9] == 0x7f && r[10] == 0xff) {
		t := float64(int32(r[9])<<24>>16|int32(r[10])) / 10
		m = append(m, Measurement{"Temp C3", t})
	}
	return m, nil
}

// mode4: weight.
type mode4 struct{}

func (mode4) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	w := float64(int64(r[9])<<24 | int64(r[10])<<16 | int64(r[7])<<8 | int64(r[8]))
	return []Measurement{{"Weight", w}}, nil
}

// mode5: counter.
type mode5 struct{}

func (mode5) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	c := float64(uint32(r[7])<<24 | uint32(r[8])<<16 | uint32(r[9])<<8 | uint32(r[10]))
	return []Measurement{{"Count", c}}, nil
}

// mode7: ADC + DS18B20.
type mode7 struct{}

func (mode7) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := commonMetrics(p)
	m = append(m,
		Measurement{"ADC CH1V", float64(int64(r[7])<<8|int64(r[8])) / 1000},
		Measurement{"ADC CH4V", float64(int64(r[9])<<8|int64(r[10])) / 1000},
	)
	return m, nil
}

// mode8: DS18B20 + 2 counters.
type mode8 struct{}

func (mode8) Decode(p *Packet) ([]Measurement, error) {
	r := p.Raw
	m := commonMetrics(p)
	if !(r[4] == 0x7f && r[5] == 0xff) {
		t := float64(int32(r[4])<<24>>16|int32(r[5])) / 10
		m = append(m, Measurement{"Temp C2", t})
	}
	if !(r[7] == 0x7f && r[8] == 0xff) {
		t := float64(int32(r[7])<<24>>16|int32(r[8])) / 10
		m = append(m, Measurement{"Temp C3", t})
	}
	c1 := float64(uint32(r[9])<<24 | uint32(r[10])<<16 | uint32(r[11])<<8 | uint32(r[12]))
	c2 := float64(uint32(r[13])<<24 | uint32(r[14])<<16 | uint32(r[15])<<8 | uint32(r[16]))
	m = append(m,
		Measurement{"Count 1", c1},
		Measurement{"Count 2", c2},
	)
	return m, nil
}
