package app

import "testing"

func TestGreeting(t *testing.T) {
	a := NewApp()
	if got := a.Greeting(); got != "gifly" {
		t.Errorf("Greeting() = %q, want gifly", got)
	}
}
