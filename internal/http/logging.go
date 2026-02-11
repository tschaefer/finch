/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package http

import (
	"context"
	"log/slog"
	"net/http"
)

func (s *Server) log(r *http.Request, level slog.Level, msg string, args ...any) {
	remoteAddr := ""
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		if v := r.Header.Get(h); len(v) > 0 && v != "" {
			remoteAddr = v
			break
		}
	}
	if remoteAddr == "" {
		remoteAddr = r.RemoteAddr
		for i := len(remoteAddr) - 1; i >= 0; i-- {
			if remoteAddr[i] == ':' {
				remoteAddr = remoteAddr[:i]
				break
			}
		}
	}
	userAgent := r.Header.Get("User-Agent")
	args = append(args, "remote_addr", remoteAddr, "user_agent", userAgent)

	ctx := context.Background()
	slog.Log(ctx, level, msg, args...)
}
