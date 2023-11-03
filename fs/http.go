package fs

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var httpClient = &http.Client{}

func httpGet(ctx context.Context, url string) (*http.Response, error) {
	ctx, span := tr.Start(ctx, "httpGet",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("url", url)))
	defer span.End()

	return httpClient.Get(url)
}
