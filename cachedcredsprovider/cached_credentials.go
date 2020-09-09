package cachedcredsprovider

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

// CachedCredentials represents the entire JSON blob that we store.
type CachedCredentials struct {
	ExpiresAt time.Time
	Values    credentials.Value
}
