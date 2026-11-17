package security

import (
	"regexp"
	"testing"
	"time"
)

func TestOTPFormat(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	for i := 0; i < 50; i++ {
		otp, err := RandomOTP()
		if err != nil || !re.MatchString(otp) {
			t.Fatalf("neispravan OTP %q (%v)", otp, err)
		}
	}
}

func TestHashTokenDeterministicAndKeyed(t *testing.T) {
	a := HashToken("secret-1", "value")
	if a != HashToken("secret-1", "value") {
		t.Error("hes mora biti deterministican")
	}
	if a == HashToken("secret-2", "value") {
		t.Error("hes mora zavisiti od tajne")
	}
}

func TestJWTRoundTrip(t *testing.T) {
	j := NewJWT("0123456789abcdef0123456789abcdef", time.Minute)
	tok, jti, _, err := j.Issue("user-1", "RK")
	if err != nil {
		t.Fatal(err)
	}
	c, err := j.Parse(tok)
	if err != nil || c.Subject != "user-1" || c.ID != jti || c.Role != "RK" {
		t.Fatalf("neocekivani claims: %+v (%v)", c, err)
	}
	if _, err := NewJWT("another-secret-another-secret-123", time.Minute).Parse(tok); err == nil {
		t.Error("token potpisan drugim kljucem mora biti odbijen")
	}
}
