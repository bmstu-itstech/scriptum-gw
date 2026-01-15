package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	pb "github.com/bmstu-itstech/scriptum-gw/gen/go/api/v2"
)

const chunkSize = 4 << 10 // 4 Kb

type UploadFileHandler struct {
	l        *slog.Logger
	fsClient pb.FileServiceClient
	timeout  time.Duration
}

func NewUploadFileHandler(l *slog.Logger, fsClient pb.FileServiceClient, timeout time.Duration) *UploadFileHandler {
	return &UploadFileHandler{l: l, fsClient: fsClient, timeout: timeout}
}

func (h *UploadFileHandler) Handle(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	const op = "UploadFileHandler.Handle"
	l := h.l.With(slog.String("op", op))

	err := r.ParseForm()
	if err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("failed to parse form: %v", err))
		return
	}

	f, header, err := r.FormFile("attachment")
	if err != nil {
		httpError(w, http.StatusBadRequest, fmt.Errorf("failed to get file 'attachment': %v", err))
		return
	}
	defer func() {
		err = f.Close()
		if err != nil {
			l.Warn("failed to close file", slog.String("error", err.Error()))
		}
	}()

	l.Info("received file", slog.Any("header", header))

	h.sendFile(w, f, header)
}

func (h *UploadFileHandler) sendFile(w http.ResponseWriter, f io.Reader, header *multipart.FileHeader) {
	const op = "UploadFileHandler.sendFile"
	l := h.l.With(slog.String("op", op))

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	stream, err := h.fsClient.Upload(ctx)
	if err != nil {
		l.Error("failed to open UploadFile stream", slog.String("error", err.Error()))
		httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	err = stream.Send(&pb.FileUploadRequest{
		Body: &pb.FileUploadRequest_Meta{
			Meta: &pb.FileMeta{
				Name: header.Filename,
			},
		},
	})
	if err != nil {
		l.Error("failed to send UploadFile request", slog.String("error", err.Error()))
		httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	buf := make([]byte, chunkSize)
	total := 0

	for {
		var n int
		n, err = f.Read(buf)
		if err != nil && err != io.EOF {
			l.Error("failed to read chunk", slog.String("error", err.Error()))
			httpError(w, http.StatusServiceUnavailable, err)
			return
		}
		if err == io.EOF {
			break
		}
		total += n
		err = stream.Send(&pb.FileUploadRequest{
			Body: &pb.FileUploadRequest_Chunk{
				Chunk: buf[:n],
			},
		})
		if err != nil {
			l.Error("failed to send Chunk", slog.String("error", err.Error()))
			httpError(w, http.StatusServiceUnavailable, err)
			return
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		l.Error("failed to close UploadFile stream", slog.String("error", err.Error()))
		httpError(w, http.StatusServiceUnavailable, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"file_id": resp.FileId,
		"size":    resp.Size,
	})
}

func httpError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": err.Error(),
	})
}
