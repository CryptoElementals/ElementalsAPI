package tokenunits

import "testing"

func TestValidateWithdrawTokenAmount(t *testing.T) {
	if err := ValidateWithdrawTokenAmount(1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWithdrawTokenAmount(MaxWithdrawTokenAmount); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWithdrawTokenAmount(0); err == nil {
		t.Fatal("expected error for zero")
	}
	if err := ValidateWithdrawTokenAmount(MaxWithdrawTokenAmount + 1); err == nil {
		t.Fatal("expected error above max")
	}
}
