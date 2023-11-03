package handlers

import (
	"net"
	"net/http"

	"github.com/rs/zerolog"
)

func ZerologWebdavLogger(logger zerolog.Logger, level zerolog.Level) func(req *http.Request, err error) {
	return func(req *http.Request, err error) {
		if err == nil {
			sendWithParams(req, logger.WithLevel(level))
		} else {
			sendWithParams(req, logger.Err(err))
		}
	}
}

func sendWithParams(req *http.Request, event *zerolog.Event) {
	event.Str("remote", getHost(req.RemoteAddr)).
		Str("agent", req.UserAgent()).
		Str("method", req.Method).
		Str("url", req.URL.Path).
		Msg("handled request")
}

func getHost(endpoint string) string {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint
	}
	return host
}
