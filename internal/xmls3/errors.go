package xmls3

import (
	"encoding/xml"
	"net/http"
)

type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	resp := ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}
	data, _ := xml.MarshalIndent(resp, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(data)
}

func WriteNotImplemented(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusNotImplemented, "NotImplemented", "Multipart upload is not supported in this version.", requestID)
}

func WriteAccessDenied(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusForbidden, "AccessDenied", "Access Denied.", requestID)
}

func WriteSignatureDoesNotMatch(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusForbidden, "SignatureDoesNotMatch", "The request signature we calculated does not match the signature you provided.", requestID)
}

func WriteNoSuchBucket(w http.ResponseWriter, requestID, bucket string) {
	WriteError(w, http.StatusNotFound, "NoSuchBucket", "The specified bucket does not exist.", requestID)
}

func WriteNoSuchKey(w http.ResponseWriter, requestID, key string) {
	WriteError(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", requestID)
}

func WriteInternalError(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusInternalServerError, "InternalError", "We encountered an internal error. Please try again.", requestID)
}

func WriteBadGateway(w http.ResponseWriter, requestID string) {
	WriteError(w, http.StatusBadGateway, "InternalError", "Upstream backend returned an error.", requestID)
}
