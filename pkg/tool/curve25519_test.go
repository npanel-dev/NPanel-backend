package tool

import "testing"

func TestCurve25519GenkeyDerivesPublicFromPrivate(t *testing.T) {
	private := "IMGZGlt-9z_W-Lg42MNGngb5DIIkiu_1M8Yu3oLboXI"

	public, normalizedPrivate, err := Curve25519Genkey(false, private)
	if err != nil {
		t.Fatalf("Curve25519Genkey returned error: %v", err)
	}

	if public != "Ondiyf3k-4Q_N2zkuodXjBgUcBCV0SKSGR9YaCww_Hw" {
		t.Fatalf("public key = %q, want derived key", public)
	}
	if normalizedPrivate != "IMGZGlt-9z_W-Lg42MNGngb5DIIkiu_1M8Yu3oLboXI" {
		t.Fatalf("private key = %q, want normalized input", normalizedPrivate)
	}
}
