package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	relayer "github.com/GoPolymarket/go-builder-relayer-client"
)

type signRequest struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Body      string `json:"body"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

func main() {
	key := firstEnv("BUILDER_API_KEY", "POLY_BUILDER_API_KEY")
	secret := firstEnv("BUILDER_SECRET", "POLY_BUILDER_SECRET")
	passphrase := firstEnv("BUILDER_PASS_PHRASE", "POLY_BUILDER_PASSPHRASE")
	if key == "" || secret == "" || passphrase == "" {
		log.Fatal("missing BUILDER_API_KEY/BUILDER_SECRET/BUILDER_PASS_PHRASE (or POLY_BUILDER_*)")
	}

	builderCfg := &relayer.BuilderConfig{
		Local: &relayer.BuilderCredentials{
			Key:        key,
			Secret:     secret,
			Passphrase: passphrase,
		},
	}

	remoteToken := os.Getenv("BUILDER_REMOTE_TOKEN")
	addr := firstEnv("REMOTE_SIGNER_ADDR", "SIGNER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	http.HandleFunc("/sign-builder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if remoteToken != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+remoteToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		var req signRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Method == "" || req.Path == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		method := strings.ToUpper(req.Method)
		body := req.Body
		headers, err := builderCfg.Headers(r.Context(), method, req.Path, &body, req.Timestamp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := map[string]string{
			relayer.HeaderPolyBuilderAPIKey:     headers.Get(relayer.HeaderPolyBuilderAPIKey),
			relayer.HeaderPolyBuilderPassphrase: headers.Get(relayer.HeaderPolyBuilderPassphrase),
			relayer.HeaderPolyBuilderSignature:  headers.Get(relayer.HeaderPolyBuilderSignature),
			relayer.HeaderPolyBuilderTimestamp:  headers.Get(relayer.HeaderPolyBuilderTimestamp),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	log.Printf("remote signer listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return val
		}
	}
	return ""
}
