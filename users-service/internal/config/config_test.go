package config

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "8080" || c.PasswordMaxAge != 60*24*time.Hour || c.PasswordMinAge != 24*time.Hour {
		t.Errorf("neocekivane podrazumevane vrednosti: %+v", c)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("PORT", "9999")
	t.Setenv("MONGO_URI", "mongodb://baza:27017")
	t.Setenv("PASSWORD_MAX_AGE", "2m")
	t.Setenv("TRUST_PROXY", "true")
	t.Setenv("CLIENT_URL", "https://klijent.local/")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "9999" || c.MongoURI != "mongodb://baza:27017" || c.PasswordMaxAge != 2*time.Minute || !c.TrustProxy {
		t.Errorf("vrednosti iz okruzenja nisu procitane: %+v", c)
	}
	if c.ClientURL != "https://klijent.local" {
		t.Errorf("zavrsna kosa crta nije uklonjena: %q", c.ClientURL)
	}
}

func TestLoadRejectsShortSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "kratko")
	if _, err := Load(); err == nil {
		t.Fatal("kratka tajna mora biti odbijena")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("OTP_TTL", "abc")
	t.Setenv("MAX_FAILED_LOGINS", "-3")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "OTP_TTL") || !strings.Contains(err.Error(), "MAX_FAILED_LOGINS") {
		t.Fatalf("ocekivana greska za obe promenljive, dobijeno: %v", err)
	}
}
