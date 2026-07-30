package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
)

func main() {
	asset := flag.String("asset", "", "asset file")
	tag := flag.String("tag", "", "release tag")
	name := flag.String("name", "app_linux_amd64", "asset name")
	ready := flag.String("ready", "", "write URL here")
	flag.Parse()
	if *asset == "" || *tag == "" {
		panic("--asset and --tag required")
	}
	data, err := os.ReadFile(*asset)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	base := "http://" + ln.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test/app/releases", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"tag_name": *tag, "draft": true, "prerelease": false, "body": "local E2E",
			"assets": []map[string]any{{"name": *name, "browser_download_url": base + "/asset", "size": len(data), "digest": "sha256:" + hex.EncodeToString(sum[:])}},
		}})
	})
	mux.HandleFunc("/asset", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, *asset) })
	if *ready != "" {
		_ = os.WriteFile(*ready, []byte(base), 0o600)
	}
	fmt.Println(base)
	if err := http.Serve(ln, mux); err != nil {
		panic(err)
	}
}
