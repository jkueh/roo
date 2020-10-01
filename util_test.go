package main

import "testing"

func TestEmptyOTP(t *testing.T) {
	if oneTimePasscodeIsValid("") {
		t.Errorf("Empty OTP passed validation")
	}
}

func TestAlphabetOTP(t *testing.T) {
	if oneTimePasscodeIsValid("hunter2") {
		t.Errorf("Invalid (alphabetical) OTP passed validation")
	}
}

func TestShortOTP(t *testing.T) {
	if oneTimePasscodeIsValid("42069") {
		t.Errorf("Invalid (short) OTP passed validation")
	}
}

func TestOTP(t *testing.T) {
	if !oneTimePasscodeIsValid("054389") {
		t.Errorf("A valid OTP did not pass validation")
	}
}
