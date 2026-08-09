package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/listbuckets"
	"github.com/egose/s3proxy/internal/replaybody"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
	"github.com/egose/s3proxy/internal/xmls3"
	"github.com/google/uuid"
)

type Dependencies struct {
	Addressing         config.Addressing
	ReplayBodyMaxBytes int64
	ReplayBudget       *replaybody.Budget
	Authenticator      auth.Authenticator
	Authorizer         auth.Authorizer
	Router             router.RouteResolver
	Rewriter           rewrite.Engine
	Dispatcher         dispatch.Fanout
	Buckets            listbuckets.Service
	Logger             *slog.Logger
}

func NewHandler(deps Dependencies) http.Handler {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.ReplayBudget == nil {
		deps.ReplayBudget = replaybody.NewBudget(deps.ReplayBodyMaxBytes, replaybody.DefaultAggregateMaxBytes)
	}
	return &handler{deps: deps}
}

type handler struct {
	deps Dependencies
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	w.Header().Set("X-Request-Id", requestID)
	logger := h.deps.Logger.With("request_id", requestID)

	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}
	if r.URL.Path == "/readyz" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

	ctx, err := requestctx.FromRequest(r, h.deps.Addressing)
	if err != nil {
		logger.Error("request parse failed", "error", err)
		if requestctx.IsNoAddressingMatch(err) {
			xmls3.WriteError(w, http.StatusBadRequest, "InvalidRequest", "The request does not match the enabled listener addressing modes.", requestID)
			return
		}
		xmls3.WriteInternalError(w, requestID)
		return
	}

	op, err := s3ops.Classify(ctx)
	if err != nil {
		logger.Error("classification failed", "error", err)
		xmls3.WriteInternalError(w, requestID)
		return
	}

	if s3ops.IsMultipart(r) {
		logger.Info("rejecting multipart", "operation", op)
		xmls3.WriteNotImplemented(w, requestID)
		return
	}
	if op == s3ops.OpUnknown || op == s3ops.OpListObjectsV1 || op == s3ops.OpCopyObject {
		xmls3.WriteNotImplemented(w, requestID)
		return
	}

	principal, err := h.deps.Authenticator.Authenticate(r)
	defer replaybody.Release(r)
	if err != nil {
		logger.Warn("auth failed", "error", err)
		if auth.IsSignatureMismatch(err) {
			xmls3.WriteSignatureDoesNotMatch(w, requestID)
			return
		}
		xmls3.WriteAccessDenied(w, requestID)
		return
	}

	if op == s3ops.OpListBuckets {
		if !h.deps.Authorizer.AllowOperation(principal, op) {
			logger.Warn("operation not authorized", "operation", op)
			xmls3.WriteAccessDenied(w, requestID)
			return
		}
		h.handleListBuckets(w, principal, requestID, logger)
		return
	}

	matches, err := h.deps.Router.Resolve(ctx, op)
	if err != nil {
		logger.Info("no route", "bucket", ctx.Bucket, "key", ctx.Key, "operation", op)
		xmls3.WriteNoSuchBucket(w, requestID, ctx.Bucket)
		return
	}

	for _, match := range matches {
		if !h.deps.Authorizer.AllowRoute(principal, match.Route.Name, op) {
			logger.Warn("route not authorized", "route", match.Route.Name)
			xmls3.WriteAccessDenied(w, requestID)
			return
		}
	}

	if len(matches) > 1 {
		if err := h.deps.ReplayBudget.Ensure(r); err != nil {
			logger.Error("request body read failed", "error", err)
			if replaybody.IsTooLarge(err) {
				xmls3.WriteError(w, http.StatusRequestEntityTooLarge, "EntityTooLarge", "Request body is too large for multi-destination replay.", requestID)
				return
			}
			if replaybody.IsBudgetExhausted(err) {
				xmls3.WriteError(w, http.StatusServiceUnavailable, "SlowDown", "Aggregate replay body memory is exhausted. Retry later.", requestID)
				return
			}
			xmls3.WriteBadGateway(w, requestID)
			return
		}
	}

	var primary *s3.Response
	for _, match := range matches {
		if len(matches) > 1 {
			if err := replaybody.Reset(r); err != nil {
				logger.Error("request body reset failed", "error", err)
				closeS3Response(primary)
				xmls3.WriteBadGateway(w, requestID)
				return
			}
		}

		rwResult, err := h.deps.Rewriter.Apply(ctx, match.Route, match.Captures)
		if err != nil {
			logger.Error("rewrite failed", "route", match.Route.Name, "error", err)
			closeS3Response(primary)
			xmls3.WriteInternalError(w, requestID)
			return
		}

		matchLogger := logger.With("route", match.Route.Name, "bucket", rwResult.Bucket, "key", rwResult.Key)
		dispResult, err := h.deps.Dispatcher.Dispatch(r.Context(), match, r, op, rwResult)
		if err != nil {
			matchLogger.Error("dispatch failed", "error", err)
			closeS3Response(primary)
			if replaybody.IsTooLarge(err) {
				xmls3.WriteError(w, http.StatusRequestEntityTooLarge, "EntityTooLarge", "Request body is too large for multi-destination replay.", requestID)
				return
			}
			if replaybody.IsBudgetExhausted(err) {
				xmls3.WriteError(w, http.StatusServiceUnavailable, "SlowDown", "Aggregate replay body memory is exhausted. Retry later.", requestID)
				return
			}
			if dispResult != nil && dispResult.Primary != nil && dispResult.Primary.StatusCode >= http.StatusBadRequest {
				if err := writeS3Response(w, dispResult.Primary); err != nil {
					matchLogger.Error("response copy failed", "error", err)
				}
				return
			}
			if dispResult != nil {
				closeS3Response(dispResult.Primary)
			}
			xmls3.WriteBadGateway(w, requestID)
			return
		}

		if dispResult != nil && dispResult.Primary != nil {
			if dispResult.Primary.StatusCode >= http.StatusBadRequest {
				closeS3Response(primary)
				if err := writeS3Response(w, dispResult.Primary); err != nil {
					matchLogger.Error("response copy failed", "error", err)
				}
				return
			}
			if primary == nil {
				primary = dispResult.Primary
			} else {
				closeS3Response(dispResult.Primary)
			}
		}
	}

	if primary != nil {
		if err := writeS3Response(w, primary); err != nil {
			logger.Error("response copy failed", "error", err)
		}
		return
	}

	xmls3.WriteInternalError(w, requestID)
}

func (h *handler) handleListBuckets(w http.ResponseWriter, principal *auth.Principal, requestID string, logger *slog.Logger) {
	views := h.deps.Buckets.List(principal)
	entries := make([]xmls3.BucketEntry, 0, len(views))
	for _, v := range views {
		entries = append(entries, xmls3.BucketEntry{
			Name:         v.Name,
			CreationDate: v.CreationDate.UTC().Format("2006-01-02T15:04:05.000Z"),
		})
	}
	ownerID := "s3proxy"
	ownerName := "s3proxy"
	if principal != nil {
		ownerID = principal.AccessKey
		ownerName = principal.Name
	}
	xmls3.WriteListBuckets(w, ownerID, ownerName, entries)
}

func closeS3Response(resp *s3.Response) {
	s3.DrainAndClose(resp)
}

func writeS3Response(w http.ResponseWriter, resp *s3.Response) error {
	connectionTokens := connectionHeaderTokens(resp.Header)
	for key, vals := range resp.Header {
		if isHopByHopHeader(key, connectionTokens) {
			continue
		}
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		_, copyErr := io.Copy(w, resp.Body)
		closeErr := resp.Body.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
func isHopByHopHeader(name string, connectionTokens map[string]struct{}) bool {
	canonical := strings.ToLower(name)
	switch canonical {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	}
	_, ok := connectionTokens[canonical]
	return ok
}

func connectionHeaderTokens(headers http.Header) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, raw := range headers.Values("Connection") {
		for _, token := range strings.Split(raw, ",") {
			token = strings.ToLower(strings.TrimSpace(token))
			if token != "" {
				tokens[token] = struct{}{}
			}
		}
	}
	return tokens
}
