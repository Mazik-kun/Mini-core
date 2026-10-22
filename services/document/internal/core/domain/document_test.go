package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestStatusTransitions(t *testing.T) {
	all := []Status{StatusPending, StatusConfirmed, StatusRejected}
	allowed := map[[2]Status]bool{
		{StatusPending, StatusConfirmed}: true,
		{StatusPending, StatusRejected}:  true,
	}
	for _, f := range all {
		for _, to := range all {
			if got := f.CanTransitionTo(to); got != allowed[[2]Status{f, to}] {
				t.Errorf("%s → %s: got %v", f, to, got)
			}
		}
	}
}

func TestDetectContentType(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	jpg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 32)...)
	pdf := []byte("%PDF-1.7\n%....")
	cases := []struct {
		name string
		in   []byte
		want string
		err  error
	}{
		{"png", png, "image/png", nil},
		{"jpeg", jpg, "image/jpeg", nil},
		{"pdf", pdf, "application/pdf", nil},
		{"text", []byte("hello world"), "text/plain", ErrUnsupportedContentType},
		{"html", []byte("<html><body>x</body></html>"), "text/html", ErrUnsupportedContentType},
	}
	for _, c := range cases {
		got, err := DetectContentType(c.in)
		if got != c.want || err != c.err {
			t.Errorf("%s: got (%q,%v) want (%q,%v)", c.name, got, err, c.want, c.err)
		}
	}
}

func TestObjectKey(t *testing.T) {
	c, d := uuid.New(), uuid.New()
	want := c.String() + "/PASSPORT/" + d.String()
	if got := ObjectKey(c, TypePassport, d); got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestParseType(t *testing.T) {
	if got, err := ParseType(" passport "); err != nil || got != TypePassport {
		t.Errorf("got %v %v", got, err)
	}
	if _, err := ParseType("VISA"); err != ErrInvalidDocumentType {
		t.Errorf("want ErrInvalidDocumentType, got %v", err)
	}
}

func TestMissing(t *testing.T) {
	req := []Type{TypePassport, TypeSelfie}
	if m := Missing(req, []Type{TypePassport, TypeSelfie, TypeProofOfAddress}); len(m) != 0 {
		t.Errorf("want none, got %v", m)
	}
	if m := Missing(req, []Type{TypePassport}); len(m) != 1 || m[0] != TypeSelfie {
		t.Errorf("want [SELFIE], got %v", m)
	}
	if m := Missing(req, nil); len(m) != 2 {
		t.Errorf("want 2, got %v", m)
	}
}
