package validate

import "testing"

func TestPassword(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		ok   bool
	}{
		{"jaka", "Str0ng!Passw0rd", true},
		{"prekratka", "Ab1!", false},
		{"bez velikog", "alllowercase1!", false},
		{"bez malog", "ALLUPPERCASE1!", false},
		{"bez cifre", "NoDigitsHere!!", false},
		{"bez specijalnog", "NoSpecial12345", false},
		{"sadrzi imejl", "Marko123!Marko", false},
		{"predugacka", "Aa1!" + string(make([]byte, 80)), false},
	}
	for _, c := range cases {
		err := Password(c.pw, "marko@example.com")
		if (err == nil) != c.ok {
			t.Errorf("%s: ocekivano ok=%v, dobijena greska=%v", c.name, c.ok, err)
		}
	}
}

func TestEmail(t *testing.T) {
	if got, err := Email("  Pera@Example.COM "); err != nil || got != "pera@example.com" {
		t.Errorf("normalizacija nije uspela: %q, %v", got, err)
	}
	for _, bad := range []string{"", "bez-et.com", "Ime <a@b.com>", "a@b", "a@b.com\r\nBcc: x@y.com"} {
		if _, err := Email(bad); err == nil {
			t.Errorf("%q je trebalo da bude odbijen", bad)
		}
	}
}

func TestName(t *testing.T) {
	if _, err := Name("Ime", "Đorđe"); err != nil {
		t.Errorf("Đorđe je trebalo da prodje: %v", err)
	}
	if _, err := Name("Ime", "<script>alert(1)</script>"); err == nil {
		t.Error("HTML u imenu je trebalo da bude odbijen")
	}
}
