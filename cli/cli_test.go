package cli

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

func writeTempConfig(t *testing.T, dir string, cfg *Config) string {
    t.Helper()
    b, _ := json.Marshal(cfg)
    p := filepath.Join(dir, "config.json")
    if err := os.WriteFile(p, b, 0o644); err != nil {
        t.Fatalf("write temp config: %v", err)
    }
    return p
}

func Test_loadConfig_PathVariants(t *testing.T) {
    dir := t.TempDir()
    // Missing
    if _, err := loadConfig(filepath.Join(dir, "nope.json")); err == nil {
        t.Fatalf("expected error for missing file")
    }

    // Valid
    cfg := &Config{Hosts: []Host{{Name: "h", URL: "http://x", Models: []string{"m"}}}}
    path := writeTempConfig(t, dir, cfg)
    got, err := loadConfig(path)
    if err != nil {
        t.Fatalf("loadConfig error: %v", err)
    }
    if len(got.Hosts) != 1 || got.Hosts[0].Name != "h" {
        t.Fatalf("unexpected cfg: %+v", got)
    }
}

func Test_getLoadedModels_SuccessAndNon200(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/ps" {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        _ = json.NewEncoder(w).Encode(map[string]any{
            "models": []map[string]any{{"name": "a"}, {"name": "b"}},
        })
    }))
    t.Cleanup(srv.Close)

    host := Host{Name: "h", URL: srv.URL}
    got, err := getLoadedModels(host, &http.Client{})
    if err != nil {
        t.Fatalf("getLoadedModels err: %v", err)
    }
    if len(got) != 2 || got[0] != "a" || got[1] != "b" {
        t.Fatalf("unexpected models: %v", got)
    }

    // Non-200
    srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusTeapot)
    }))
    t.Cleanup(srvBad.Close)
    _, err = getLoadedModels(Host{Name: "h", URL: srvBad.URL}, &http.Client{})
    if err == nil {
        t.Fatalf("expected error on non-200 status")
    }
}

func Test_loadModelCmd_SuccessAndError(t *testing.T) {
    // Success server
    ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/generate" {
            w.WriteHeader(http.StatusOK)
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    t.Cleanup(ok.Close)

    // Execute command
    msg := loadModelCmd(Host{Name: "h", URL: ok.URL}, "m1", &http.Client{})()
    if _, ok := msg.(chatReadyMsg); !ok {
        t.Fatalf("expected chatReadyMsg, got %T", msg)
    }

    // Error server
    bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/generate" {
            http.Error(w, "boom", http.StatusBadRequest)
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    t.Cleanup(bad.Close)

    msg = loadModelCmd(Host{Name: "h", URL: bad.URL}, "m1", &http.Client{})()
    if _, ok := msg.(chatReadyErr); !ok {
        t.Fatalf("expected chatReadyErr, got %T", msg)
    }
}

func Test_formatMeta_StringContainsValues(t *testing.T) {
    meta := LLMResponseMeta{
        LoadDuration:       int64(1.2 * float64(time.Second)),
        PromptEvalDuration: int64(0.5 * float64(time.Second)),
        PromptEvalCount:    10,
        EvalDuration:       int64(0.8 * float64(time.Second)),
        EvalCount:          20,
        TotalDuration:      int64(2.5 * float64(time.Second)),
    }
    s := formatMeta(meta)
    for _, want := range []string{"1.2s", "0.5s", "0.8s", "2.5s", "10", "20"} {
        if !strings.Contains(s, strings.TrimSuffix(want, "s")) { // rendered values may be rounded
            t.Fatalf("expected %q in %q", want, s)
        }
    }
}

