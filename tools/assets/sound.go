package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"os"
	"path/filepath"
)

const rate = 44100

// A sound is built up as float samples in -1..1 and written as 16-bit mono.
type sound []float64

func newSound(seconds float64) sound { return make(sound, int(seconds*rate)) }

// tone adds a note that starts at `at` seconds. Its pitch slides from f0 to f1
// and its volume rises over `attack` seconds, then dies away with time
// constant `decay`. partials are the levels of the 2nd, 3rd... harmonics,
// which give bell-like notes some body.
func (s sound) tone(at, length, f0, f1, level, attack, decay float64, partials ...float64) {
	start := int(at * rate)
	n := int(length * rate)
	phase := 0.0
	for i := 0; i < n && start+i < len(s); i++ {
		t := float64(i) / rate
		f := f0 * math.Pow(f1/f0, t/length)
		phase += 2 * math.Pi * f / rate
		v := math.Sin(phase)
		for k, p := range partials {
			v += p * math.Sin(phase*float64(k+2))
		}
		s[start+i] += v * level * envelope(t, length, attack, decay)
	}
}

// noise adds low-passed noise whose cutoff falls from c0 to c1 Hz.
func (s sound) noise(seed int64, at, length, c0, c1, level, attack, decay float64) {
	rng := rand.New(rand.NewSource(seed))
	start := int(at * rate)
	n := int(length * rate)
	y := 0.0
	for i := 0; i < n && start+i < len(s); i++ {
		t := float64(i) / rate
		cut := c0 * math.Pow(c1/c0, t/length)
		a := 1 - math.Exp(-2*math.Pi*cut/rate)
		y += a * (rng.Float64()*2 - 1 - y)
		s[start+i] += y * level * envelope(t, length, attack, decay)
	}
}

func envelope(t, length, attack, decay float64) float64 {
	e := math.Exp(-t / decay)
	if attack > 0 && t < attack {
		e *= t / attack
	}
	// a short fade at the very end so nothing clicks
	if tail := length - t; tail < 0.01 {
		e *= tail / 0.01
	}
	return e
}

func (s sound) save(dir, name string) {
	data := make([]byte, 44+len(s)*2)
	copy(data[0:], "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+len(s)*2))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1) // PCM
	binary.LittleEndian.PutUint16(data[22:], 1) // mono
	binary.LittleEndian.PutUint32(data[24:], rate)
	binary.LittleEndian.PutUint32(data[28:], rate*2)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(len(s)*2))
	for i, v := range s {
		v = math.Tanh(v) // soft clip
		binary.LittleEndian.PutUint16(data[44+i*2:], uint16(int16(math.Round(v*32767))))
	}
	must(os.WriteFile(filepath.Join(dir, name+".wav"), data, 0o644))
}

// note returns the frequency of a MIDI note number (69 is A4).
func note(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }

var bell = []float64{0.25, 0.08}

func buildSounds(dir string) {
	s := newSound(0.09)
	s.tone(0, 0.09, 1250, 720, 0.28, 0.002, 0.03)
	s.noise(1, 0, 0.03, 6000, 2000, 0.08, 0, 0.01)
	s.save(dir, "shoot")

	s = newSound(0.14)
	s.tone(0, 0.14, 520, 300, 0.3, 0.003, 0.05, 0.3)
	s.save(dir, "enemy-shoot")

	s = newSound(0.08)
	s.noise(2, 0, 0.08, 4000, 900, 0.5, 0, 0.02)
	s.tone(0, 0.06, 700, 500, 0.15, 0, 0.02)
	s.save(dir, "hit")

	s = newSound(0.6)
	s.noise(3, 0, 0.6, 2400, 140, 0.9, 0.002, 0.14)
	s.tone(0, 0.35, 120, 45, 0.5, 0.004, 0.09)
	s.save(dir, "explode")

	s = newSound(1.8)
	s.noise(4, 0, 1.8, 1800, 80, 1.0, 0.004, 0.45)
	s.tone(0, 1.0, 90, 30, 0.6, 0.006, 0.3)
	s.noise(5, 0.22, 0.8, 2600, 150, 0.6, 0.002, 0.18)
	s.noise(6, 0.5, 0.9, 2000, 120, 0.45, 0.002, 0.22)
	s.save(dir, "boss-explode")

	s = newSound(0.5)
	s.tone(0, 0.5, 520, 130, 0.4, 0.003, 0.16, 0.2)
	s.noise(7, 0, 0.3, 1500, 200, 0.35, 0.002, 0.08)
	s.save(dir, "hurt")

	s = newSound(0.5)
	s.tone(0, 0.5, note(79), note(79), 0.22, 0.004, 0.12, bell...)
	s.tone(0.07, 0.43, note(86), note(86), 0.22, 0.004, 0.14, bell...)
	s.save(dir, "powerup")

	s = newSound(0.9)
	for i, n := range []float64{72, 76, 79} {
		s.tone(float64(i)*0.09, 0.9-float64(i)*0.09, note(n), note(n), 0.2, 0.005, 0.22, bell...)
	}
	s.save(dir, "start")

	s = newSound(0.7)
	s.tone(0, 0.7, note(67), note(67), 0.2, 0.006, 0.2, bell...)
	s.tone(0.12, 0.58, note(74), note(74), 0.2, 0.006, 0.22, bell...)
	s.save(dir, "wave")

	s = newSound(1.6)
	for i, n := range []float64{64, 60, 57} {
		at := float64(i) * 0.26
		s.tone(at, 1.6-at, note(n), note(n), 0.24, 0.01, 0.35+float64(i)*0.15, bell...)
	}
	s.save(dir, "game-over")
}
