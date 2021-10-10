package http

import (
	"context"
	"time"

	"github.com/medicplus-inc/medicplus-kit/net/http/middleware"
	"github.com/medicplus-inc/medicplus-kit/net/structure"
)

func ResponseError(
	statusCode int,
	error error,
	message string,
) structure.RejectStructure {
	return structure.RejectStructure{
		Code:    statusCode,
		Error:   error,
		Message: message,
	}
}

func ResponseWithRequestTime(
	ctx context.Context,
	data interface{},
	metadata map[string]interface{},
) interface{} {
	meta := make(map[string]interface{})

	for k, v := range metadata {
		meta[k] = v
	}

	if value := ctx.Value(middleware.KEY_REQUEST_TIME); value != nil {
		meta["request_took"] = time.Since(value.(time.Time)).Seconds()
		meta["request_measure"] = "time_measure.second"
	}

	return structure.ResolveStructure{
		Data:     data,
		Metadata: meta,
	}
}
