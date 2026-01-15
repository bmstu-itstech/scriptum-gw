package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/bmstu-itstech/scriptum-gw/gen/go/api/v2"
	"github.com/bmstu-itstech/scriptum-gw/internal/config"
	"github.com/bmstu-itstech/scriptum-gw/pkg/auth"
	"github.com/bmstu-itstech/scriptum-gw/pkg/logs"
)

func errorHandler(
	_ context.Context, _ *runtime.ServeMux, _ runtime.Marshaler, w http.ResponseWriter, _ *http.Request, err error,
) {
	st := status.Convert(err)

	httpStatus := runtime.HTTPStatusFromCode(st.Code())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	switch httpStatus {
	case http.StatusUnauthorized:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "missing or invalid 'Authorization: Bearer ...'",
		})
	case http.StatusInternalServerError:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "internal server error",
		})
	default:
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": st.Message(),
		})
	}
}

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "path to config file")
	flag.Parse()
	if cfgPath == "" {
		flag.Usage()
		os.Exit(1)
	}
	cfg := config.MustLoad(cfgPath)
	l := logs.NewLogger(cfg.Logging)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDisableRetry(),
	}

	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(errorHandler),
		runtime.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
			if uid, ok := auth.ExtractUIDFromContext(r.Context()); ok {
				return metadata.Pairs("x-user-id", strconv.FormatInt(uid, 10))
			}
			return nil
		}),
		runtime.WithMiddlewares(
			auth.NewMiddleware(cfg.Server.JwtSecret).Handler,
			logs.NewMiddleware(l).Handler,
		),
	)

	cc, err := grpc.NewClient(cfg.FileService.Addr, opts...)
	if err != nil {
		panic(err)
	}
	fsClient := pb.NewFileServiceClient(cc)

	err = mux.HandlePath("POST", "/v2/files", NewUploadFileHandler(l, fsClient, 5*time.Second).Handle)
	if err != nil {
		panic(err)
	}

	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", mux))

	err = pb.RegisterBoxServiceHandlerFromEndpoint(ctx, mux, cfg.BoxesService.Addr, opts)
	if err != nil {
		l.Error("failed to register BoxServiceHandler",
			slog.String("error", err.Error()),
			slog.String("addr", cfg.BoxesService.Addr),
		)
		os.Exit(1)
	}

	err = pb.RegisterJobServiceHandlerFromEndpoint(ctx, mux, cfg.JobsService.Addr, opts)
	if err != nil {
		l.Error("failed to register JobServiceHandler",
			slog.String("error", err.Error()),
			slog.String("addr", cfg.JobsService.Addr),
		)
		os.Exit(1)
	}

	l.Info("Starting HTTP server")
	if err = http.ListenAndServe(cfg.Server.Addr, root); err != nil {
		l.Error("failed to start HTTP server",
			slog.String("error", err.Error()),
			slog.String("addr", cfg.Server.Addr),
		)
		os.Exit(1)
	}
}
