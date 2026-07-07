package pristinecss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// Key to use when setting the request ID in the context
type key int

const cssContextIDKey key = 0

// CSSContextIDMiddleware is a middleware that adds an ID to the context of each request for styles.
func CSSContextIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID, err := generateRandomID(16) // Generates a 32-character hex string (16 bytes)
        if err != nil {
            http.Error(w, "Failed to generate request ID", http.StatusInternalServerError)
            return
        }
        ctx := context.WithValue(r.Context(), cssContextIDKey, requestID)
        r = r.WithContext(ctx)
        next.ServeHTTP(w, r)
    })
}

// fromContext retrieves the request ID from the context
func fromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(cssContextIDKey).(string)
	return requestID, ok
}

// generateRandomID generates a cryptographically secure random ID.
func generateRandomID(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err // return an empty string and the error
    }
    return hex.EncodeToString(bytes), nil
}

