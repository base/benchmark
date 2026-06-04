package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	resultsDir := flag.String("results-dir", "./results", "directory containing benchmark results")
	port := flag.String("port", "8080", "HTTP listen port")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/output/metadata.json", metadataHandler(*resultsDir))
	mux.HandleFunc("/output/", fileHandler(*resultsDir))

	addr := ":" + *port
	log.Printf("listening on %s, serving results from %s", addr, *resultsDir)
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func metadataHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(resultsDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var allRuns []json.RawMessage

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			metaPath := filepath.Join(resultsDir, entry.Name(), "metadata.json")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var meta struct {
				Runs []json.RawMessage `json:"runs"`
			}
			if err := json.Unmarshal(data, &meta); err != nil {
				continue
			}
			if len(meta.Runs) > 0 {
				allRuns = append(allRuns, meta.Runs[0])
			}
		}

		if allRuns == nil {
			allRuns = []json.RawMessage{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"runs": allRuns,
		})
	}
}

func fileHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/output/")
		if relPath == "" || strings.Contains(relPath, "..") {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		filePath := filepath.Join(resultsDir, relPath)
		f, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			http.Error(w, "not a file", http.StatusBadRequest)
			return
		}

		http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
	}
}
