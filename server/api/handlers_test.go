package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postBody(t *testing.T, payload any) *strings.Reader {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return strings.NewReader(string(b))
}

func readJSONError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var m map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return m["error"]
}

func newTestAPI() *API {
	return &API{
		cfg:     nil,
		srv:     nil,
		blocker: nil,
		stats:   nil,
		cache:   nil,
	}
}

// ---------------------------------------------------------------------------
// handleHostsAdd – validation
// ---------------------------------------------------------------------------

func TestHandleHostsAdd_InvalidIP(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/add", postBody(t, map[string]string{
		"domain": "test.local",
		"ip":     "not-an-ip",
	}))

	api.handleHostsAdd(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := readJSONError(t, rec); msg != "invalid IP address" {
		t.Errorf("error = %q, want %q", msg, "invalid IP address")
	}
}

func TestHandleHostsAdd_EmptyDomain(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/add", postBody(t, map[string]string{
		"domain": "",
		"ip":     "192.168.1.1",
	}))

	api.handleHostsAdd(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := readJSONError(t, rec); msg != "domain and ip are required" {
		t.Errorf("error = %q, want %q", msg, "domain and ip are required")
	}
}

func TestHandleHostsAdd_EmptyIP(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/add", postBody(t, map[string]string{
		"domain": "test.local",
		"ip":     "",
	}))

	api.handleHostsAdd(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleHostsAdd_InvalidJSON(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/add", strings.NewReader("{bad json"))

	api.handleHostsAdd(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleHostsAdd_WrongMethod(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/add", nil)

	api.handleHostsAdd(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// handleHostsRemove – validation
// ---------------------------------------------------------------------------

func TestHandleHostsRemove_EmptyDomain(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/remove", postBody(t, map[string]string{
		"domain": "",
	}))

	api.handleHostsRemove(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := readJSONError(t, rec); msg != "domain is required" {
		t.Errorf("error = %q, want %q", msg, "domain is required")
	}
}

func TestHandleHostsRemove_InvalidJSON(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/remove", strings.NewReader("not-json"))

	api.handleHostsRemove(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleHostsRemove_WrongMethod(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/remove", nil)

	api.handleHostsRemove(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// mutateBlocklistDomain – validation
// ---------------------------------------------------------------------------

func TestMutateBlocklist_EmptyDomain(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/blocklist/add", postBody(t, map[string]string{
		"domain": "",
	}))

	api.mutateBlocklistDomain(rec, req, true, "added")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if msg := readJSONError(t, rec); msg != "domain is required" {
		t.Errorf("error = %q, want %q", msg, "domain is required")
	}
}

func TestMutateBlocklist_WhitespaceOnlyDomain(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/blocklist/add", postBody(t, map[string]string{
		"domain": "   ",
	}))

	api.mutateBlocklistDomain(rec, req, true, "added")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMutateBlocklist_InvalidJSON(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/blocklist/add", strings.NewReader("{invalid"))

	api.mutateBlocklistDomain(rec, req, true, "added")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMutateBlocklist_WrongMethod(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/blocklist/add", nil)

	api.mutateBlocklistDomain(rec, req, true, "added")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// handleBlockPageConfig – validation
// ---------------------------------------------------------------------------

func TestHandleBlockPageConfig_InvalidJSON(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config/block-page", strings.NewReader("not-json"))

	api.handleBlockPageConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleBlockPageConfig_WrongMethod(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/config/block-page", nil)

	api.handleBlockPageConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// handleConfig – validation
// ---------------------------------------------------------------------------

func TestHandleConfig_InvalidJSON(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader("bad"))

	api.handleConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleConfig_WrongMethod(t *testing.T) {
	api := newTestAPI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/config", nil)

	api.handleConfig(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// ---------------------------------------------------------------------------
// isLoopback – helper validation
// ---------------------------------------------------------------------------

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"127.255.255.255", true},
		{"::1", true},
		{"0.0.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"1.2.3.4", false},
		{"invalid-host", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			if got := isLoopback(tc.host); got != tc.want {
				t.Errorf("isLoopback(%q) = %v, want %v", tc.host, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IP validation (net.ParseIP) – standalone helper tests
// ---------------------------------------------------------------------------

func TestIPValidation(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"1.1.1.1",
		"255.255.255.255",
		"0.0.0.0",
		"::1",
		"2001:db8::1",
	}
	for _, ip := range validIPs {
		t.Run("valid/"+ip, func(t *testing.T) {
			if net.ParseIP(ip) == nil {
				t.Errorf("net.ParseIP(%q) returned nil, expected valid", ip)
			}
		})
	}

	invalidIPs := []string{
		"not-an-ip",
		"256.0.0.1",
		"192.168.1",
		"192.168.1.1.5",
		":::1",
		"abc.def.ghi.jkl",
		"",
	}
	for _, ip := range invalidIPs {
		t.Run("invalid/"+ip, func(t *testing.T) {
			if net.ParseIP(ip) != nil {
				t.Errorf("net.ParseIP(%q) returned non-nil, expected nil", ip)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// net.SplitHostPort validation – used for upstream config
// ---------------------------------------------------------------------------

func TestSplitHostPortValidation(t *testing.T) {
	valid := []string{
		"127.0.0.1:53",
		"[::1]:53",
		"8.8.8.8:853",
		":8080",
	}
	for _, addr := range valid {
		t.Run("valid/"+addr, func(t *testing.T) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				t.Fatalf("SplitHostPort(%q) error: %v", addr, err)
			}
			if host == "" && port == "" {
				t.Error("both host and port empty")
			}
		})
	}

	invalid := []string{
		"127.0.0.1",
		"no-port",
		"",
	}
	for _, addr := range invalid {
		t.Run("invalid/"+addr, func(t *testing.T) {
			_, _, err := net.SplitHostPort(addr)
			if err == nil {
				t.Errorf("SplitHostPort(%q) expected error, got nil", addr)
			}
		})
	}
}
