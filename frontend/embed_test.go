// Guards the embedded UI: the SPA at / and the preserved admin at /admin/.
package frontend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSPAAndAdmin(t *testing.T) {
	handler := Handler()
	cases := []struct {
		path     string
		contains string
	}{
		{"/", `id="app"`},
		{"/admin/", "temporary admin"},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d, want 200", tc.path, recorder.Code)
		}
		body, _ := io.ReadAll(recorder.Result().Body)
		if !strings.Contains(string(body), tc.contains) {
			t.Errorf("GET %s: body missing %q", tc.path, tc.contains)
		}
	}
}
