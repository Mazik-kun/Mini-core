package password

import (

	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
   	s, err := HashPassword("hello")
	if err!= nil{
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(s,"$argon2id$v=19$m=65536,t=3,p=4$"){
		t.Errorf("wrond prefix: %q", s)
	}
}
func TestVerifyPassword(t *testing.T) {
	s, err := HashPassword("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		ok, err := VerifyPassword("hello", s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		ok, err := VerifyPassword("hamlo", s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Errorf("expected false, got true")
		}
	})

	t.Run("broken hash", func(t *testing.T) {
		_, err := VerifyPassword("hello", "not-a-phc-string")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("base64 garbage", func(t *testing.T) {
		_, err := VerifyPassword("hello", "$argon2id$v=19$m=65536,t=3,p=4$AAAA$BBBB")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}