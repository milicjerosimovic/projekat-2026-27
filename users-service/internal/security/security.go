package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var dummyHash string

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	if err != nil {
		panic(err)
	}
	dummyHash = string(h)
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func BurnPasswordCheck(pw string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(pw))
}

func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func RandomOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func HashToken(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

func (j *JWT) Issue(userID, role string) (token, jti string, exp time.Time, err error) {
	jti, err = RandomToken(16)
	if err != nil {
		return "", "", time.Time{}, err
	}
	now := time.Now().UTC()
	exp = now.Add(j.ttl)
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return token, jti, exp, nil
}

func (j *JWT) Parse(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return j.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !t.Valid || claims.ID == "" || claims.Subject == "" {
		return nil, errors.New("neispravan token")
	}
	return claims, nil
}
