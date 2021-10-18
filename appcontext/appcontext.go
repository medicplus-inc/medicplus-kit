package appcontext

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	// KeyApplicationID represents the application id key
	KeyApplicationID contextKey = "ApplicationID"

	// KeyURLPath represents the url path key in http server context
	KeyURLPath contextKey = "URLPath"

	// KeyHTTPMethodName represents the method name key in http server context
	KeyHTTPMethodName contextKey = "HTTPMethodName"

	// KeySessionID represents the current logged-in SessionID
	KeySessionID contextKey = "SessionID"

	// KeyUserID represents the current logged-in UserID
	KeyUserID contextKey = "UserID"

	// KeyLoginToken represents the current logged-in token
	KeyLoginToken contextKey = "LoginToken"

	// KeyClientID represents the Current Client in http server context
	KeyClientID contextKey = "ClientID"

	// KeyCurrentXAccessToken represents the current access token of request
	KeyCurrentXAccessToken contextKey = "CurrentAccessToken"
)

// ApplicationID gets the application id from the context
func ApplicationID(ctx *context.Context) uuid.UUID {
	applicationID := (*ctx).Value(KeyApplicationID)
	if applicationID != nil {
		v := applicationID.(string)
		appID, err := uuid.FromBytes([]byte(v))
		if err != nil {
			return uuid.Nil
		}

		return appID
	}
	return uuid.Nil
}

// URLPath gets the data url path from the context
func URLPath(ctx *context.Context) *string {
	urlPath := (*ctx).Value(KeyURLPath)
	if urlPath != nil {
		v := urlPath.(string)
		return &v
	}
	return nil
}

// HTTPMethodName gets the data http method from the context
func HTTPMethodName(ctx *context.Context) *string {
	httpMethodName := (*ctx).Value(KeyHTTPMethodName)
	if httpMethodName != nil {
		v := httpMethodName.(string)
		return &v
	}
	return nil
}

// SessionID gets the data session id from the context
func SessionID(ctx *context.Context) *string {
	sessionID := (*ctx).Value(KeySessionID)
	if sessionID != nil {
		v := sessionID.(string)
		return &v
	}
	return nil
}

// UserID gets current userId logged in from the context
func UserID(ctx *context.Context) *int {
	userID := (*ctx).Value(KeyUserID)
	if userID != nil {
		v := userID.(int)
		return &v
	}
	return nil
}

// ClientID gets current client from the context
func ClientID(ctx *context.Context) *int {
	currentClientAccess := (*ctx).Value(KeyClientID)
	if currentClientAccess != nil {
		v := currentClientAccess.(int)
		return &v
	}
	return nil
}

// CurrentXAccessToken gets current x access token code of request
func CurrentXAccessToken(ctx *context.Context) string {
	currentAccessToken := (*ctx).Value(KeyCurrentXAccessToken)
	if currentAccessToken != nil {
		v := currentAccessToken.(string)
		return v
	}
	return ""
}
