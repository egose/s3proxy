package httpapi

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/egose/s3proxy/internal/auth"
	"github.com/egose/s3proxy/internal/backend/s3"
	"github.com/egose/s3proxy/internal/config"
	"github.com/egose/s3proxy/internal/dispatch"
	"github.com/egose/s3proxy/internal/listbuckets"
	"github.com/egose/s3proxy/internal/requestctx"
	"github.com/egose/s3proxy/internal/rewrite"
	"github.com/egose/s3proxy/internal/router"
	"github.com/egose/s3proxy/internal/s3ops"
	"github.com/egose/s3proxy/internal/xmls3"
	"github.com/google/uuid"
)

type Dependencies struct {
	Addressing    config.Addressing
	Authenticator auth.Authenticator
	Authorizer    auth.Authorizer
	Router        router.RouteResolver
	Rewriter      rewrite.Engine
	Dispatcher    dispatch.Fanout
	Buckets       listbuckets.Service
	Logger        *slog.Logger
}

func NewHandler(deps Dependencies) http.Handler {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
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
	if op == s3ops.OpUnknown {
		xmls3.WriteNotImplemented(w, requestID)
		return
	}

	principal, err := h.deps.Authenticator.Authenticate(r)
	if err != nil {
		logger.Warn("auth failed", "error", err)
		xmls3.WriteAccessDenied(w, requestID)
		return
	}

	if op == s3ops.OpListBuckets {
		h.handleListBuckets(w, principal, requestID, logger)
		return
	}

	matches, err := h.deps.Router.Resolve(ctx, op)
	if err != nil {
		logger.Info("no route", "bucket", ctx.Bucket, "key", ctx.Key, "operation", op)
		xmls3.WriteNoSuchBucket(w, requestID, ctx.Bucket)
		return
	}
	match := matches[0]

	if !h.deps.Authorizer.AllowRoute(principal, match.Route.Name, op) {
		logger.Warn("route not authorized", "route", match.Route.Name)
		xmls3.WriteAccessDenied(w, requestID)
		return
	}

	rwResult, err := h.deps.Rewriter.Apply(ctx, match.Route, match.Captures)
	if err != nil {
		logger.Error("rewrite failed", "error", err)
		xmls3.WriteInternalError(w, requestID)
		return
	}

	logger = logger.With("route", match.Route.Name, "bucket", rwResult.Bucket, "key", rwResult.Key)

	dispResult, err := h.deps.Dispatcher.Dispatch(r.Context(), match, r, op, rwResult)
	if err != nil {
		logger.Error("dispatch failed", "error", err)
		// If the dispatcher produced a Primary response (typical for fan-out
		// partial-success), surface the upstream's actual status + body so
		// clients see MinIO/SeaweedFS error XML rather than a generic 502.
		if dispResult != nil && dispResult.Primary != nil {
			writeS3Response(w, dispResult.Primary)
			return
		}
		xmls3.WriteBadGateway(w, requestID)
		return
	}

	if dispResult.Primary != nil {
		writeS3Response(w, dispResult.Primary)
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

func writeS3Response(w http.ResponseWriter, resp *s3.Response) {
	for key, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		io.Copy(w, resp.Body)
		resp.Body.Close()
	}
}
