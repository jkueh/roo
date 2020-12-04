package main

// SigninTokenResponse - The payload that comes back from https://signin.aws.amazon.com/federation
type SigninTokenResponse struct {
	SigninToken string `json:"SigninToken"`
}
