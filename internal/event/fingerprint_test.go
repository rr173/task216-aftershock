package event

import (
	"testing"
	"time"
)

func TestFingerprint_Idempotent(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	in := EventInput{
		OriginTime: base, Latitude: 36.0, Longitude: 140.0,
		DepthKm: 12.0, Magnitude: 5.5, LocErrorH: 2, LocErrorZ: 3,
	}
	if Fingerprint(in) != Fingerprint(in) {
		t.Fatal("fingerprint must be deterministic")
	}
}

func TestFingerprint_SensitiveToFields(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	a := EventInput{OriginTime: base, Latitude: 36.0, Longitude: 140.0, DepthKm: 12, Magnitude: 5.5}
	b := a
	b.Magnitude = 5.6
	if Fingerprint(a) == Fingerprint(b) {
		t.Fatal("magnitude change must alter fingerprint")
	}
	c := a
	c.Latitude = 36.1
	if Fingerprint(a) == Fingerprint(c) {
		t.Fatal("latitude change must alter fingerprint")
	}
}

func TestValidate(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		in   EventInput
		ok   bool
	}{
		{"valid", EventInput{OriginTime: base, Latitude: 36, Longitude: 140, DepthKm: 12, Magnitude: 5.5}, true},
		{"zero time", EventInput{Latitude: 36, Longitude: 140, DepthKm: 12, Magnitude: 5.5}, false},
		{"bad latitude", EventInput{OriginTime: base, Latitude: 91, Longitude: 140, DepthKm: 12, Magnitude: 5.5}, false},
		{"bad longitude", EventInput{OriginTime: base, Latitude: 36, Longitude: 181, DepthKm: 12, Magnitude: 5.5}, false},
		{"negative depth", EventInput{OriginTime: base, Latitude: 36, Longitude: 140, DepthKm: -1, Magnitude: 5.5}, false},
		{"bad magnitude", EventInput{OriginTime: base, Latitude: 36, Longitude: 140, DepthKm: 12, Magnitude: 12}, false},
		{"negative loc error", EventInput{OriginTime: base, Latitude: 36, Longitude: 140, DepthKm: 12, Magnitude: 5.5, LocErrorH: -1}, false},
	}
	for _, c := range cases {
		err := Validate(c.in)
		if c.ok && err != nil {
			t.Errorf("%s: expected valid, got %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: expected invalid, got nil", c.name)
		}
	}
}

func TestLocUnstable(t *testing.T) {
	in := EventInput{LocErrorH: 60, LocErrorZ: 10}
	if !LocUnstable(in, 50, 40) {
		t.Fatal("expected unstable due to horizontal error")
	}
	in2 := EventInput{LocErrorH: 10, LocErrorZ: 10}
	if LocUnstable(in2, 50, 40) {
		t.Fatal("expected stable")
	}
}
