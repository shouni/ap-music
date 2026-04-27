package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHomeRendersComposeForm(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	if !strings.Contains(rec.Body.String(), `name="compose_mode"`) {
		t.Fatalf("response body does not contain compose form")
	}
}

func TestEnqueueTaskRejectsCrossOriginRequest(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	form := url.Values{
		"text": {"hello"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/compose", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	h.EnqueueTask(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestEnqueueTaskRejectsEmptySubmission(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	form := url.Values{}
	req := httptest.NewRequest(http.MethodPost, "/web/compose", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	h.EnqueueTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
