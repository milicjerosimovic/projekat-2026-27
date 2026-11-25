package handler

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	l := newRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("zahtev %d je trebalo da prodje", i+1)
		}
	}
	if ok, wait := l.allow("a"); ok || wait <= 0 {
		t.Fatal("cetvrti zahtev je trebalo da bude odbijen")
	}
	if ok, _ := l.allow("b"); !ok {
		t.Fatal("drugi kljuc ne sme da bude blokiran")
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	l := newRateLimiter(1, 20*time.Millisecond)
	if ok, _ := l.allow("a"); !ok {
		t.Fatal("prvi zahtev mora da prodje")
	}
	if ok, _ := l.allow("a"); ok {
		t.Fatal("drugi zahtev mora da bude odbijen")
	}
	time.Sleep(30 * time.Millisecond)
	if ok, _ := l.allow("a"); !ok {
		t.Fatal("posle isteka prozora zahtev mora da prodje")
	}
}
