package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var nameRe = regexp.MustCompile(`^[\p{L}][\p{L} .'\-]*$`)

var commonPasswords = map[string]bool{
	"password123!": true, "qwerty123456": true, "1234567890aA!": true,
	"welcome12345": true, "letmein12345!": true, "admin123456!": true,
}

func Name(field, v string) (string, error) {
	v = strings.TrimSpace(v)
	n := utf8.RuneCountInString(v)
	if n < 1 || n > 50 {
		return "", fmt.Errorf("%s mora imati izmedju 1 i 50 karaktera", field)
	}
	if !nameRe.MatchString(v) {
		return "", fmt.Errorf("%s sme da sadrzi samo slova, razmak, tacku, apostrof i crticu", field)
	}
	return v, nil
}

func Email(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	errBad := errors.New("Imejl adresa nije ispravna")
	if v == "" || len(v) > 254 {
		return "", errBad
	}
	addr, err := mail.ParseAddress(v)
	if err != nil || addr.Address != v {
		return "", errBad
	}
	at := strings.LastIndex(v, "@")
	if at < 1 || !strings.Contains(v[at+1:], ".") {
		return "", errBad
	}
	return v, nil
}

func Password(pw, email string) error {
	if utf8.RuneCountInString(pw) < 10 {
		return errors.New("Lozinka mora imati najmanje 10 karaktera")
	}
	if len(pw) > 72 {
		return errors.New("Lozinka je predugacka (najvise 72 bajta)")
	}

	var upper, lower, digit, special bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			special = true
		}
	}
	var missing []string
	if !upper {
		missing = append(missing, "veliko slovo")
	}
	if !lower {
		missing = append(missing, "malo slovo")
	}
	if !digit {
		missing = append(missing, "cifru")
	}
	if !special {
		missing = append(missing, "specijalni karakter")
	}
	if len(missing) > 0 {
		return fmt.Errorf("Lozinka mora sadrzati: %s", strings.Join(missing, ", "))
	}

	lowerPw := strings.ToLower(pw)
	if local, _, ok := strings.Cut(email, "@"); ok && len(local) >= 3 && strings.Contains(lowerPw, strings.ToLower(local)) {
		return errors.New("Lozinka ne sme sadrzati deo imejl adrese")
	}
	if commonPasswords[lowerPw] {
		return errors.New("Lozinka je previse cesta")
	}
	return nil
}
