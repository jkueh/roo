package main

import "testing"

func TestEmptyOTP(t *testing.T) {
	if valid, err := oneTimePasscodeIsValid(""); err == nil {
		t.Errorf("Empty OTP triggered did not trigger error as expected: %s", err)
	} else if valid {
		t.Errorf("Empty OTP passed validation")
	}
}

func TestAlphabetOTP(t *testing.T) {
	if valid, err := oneTimePasscodeIsValid("hunter2"); err == nil {
		t.Errorf("Invalid (alphabetical) OTP did not trigger error as expected: %s", err)
	} else if valid {
		t.Errorf("Invalid (alphabetical) OTP passed validation")
	}
}

func TestShortOTP(t *testing.T) {
	if valid, err := oneTimePasscodeIsValid("42069"); err == nil {
		t.Errorf("Invalid (short) OTP did not trigger error as expected: %s", err)
	} else if valid {
		t.Errorf("Invalid (short) OTP passed validation")
	}
}

func TestOTP(t *testing.T) {
	if valid, err := oneTimePasscodeIsValid("054389"); err != nil {
		t.Errorf("A valid OTP threw an error: %s", err)
	} else if !valid {
		t.Errorf("A valid OTP did not pass validation")
	}
}
