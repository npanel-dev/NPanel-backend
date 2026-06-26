package phone

import "testing"

func TestCheckWithAreaCode(t *testing.T) {
	t.Parallel()

	if !Check("1", "5123456789") {
		t.Fatal("expected valid US phone number to pass")
	}
	if Check("1", "999") {
		t.Fatal("expected short US phone number to fail")
	}
}
