package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoveryAllPagesAndThreeStates(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if q.Get("includePermissionTemplateRoles") != "true" || q.Get("maxResults") != "100" || r.Header.Get("Authorization") != "Bearer test" {
			t.Errorf("invalid request")
		}
		if calls == 1 {
			if _, exists := q["nextToken"]; exists {
				t.Error("first page must omit nextToken")
			}
			fmt.Fprint(w, `{"entities":[{"cloudAccountRoleId":"a","createdByPermissionTemplate":true},{"cloudAccountRoleId":"b","createdByPermissionTemplate":false}],"totalCount":4,"nextToken":"NT+/=&"}`)
		} else {
			if q.Get("nextToken") != "NT+/=&" {
				t.Error("cursor was not URL encoded")
			}
			fmt.Fprint(w, `{"entities":[{"cloudAccountRoleId":"c","createdByPermissionTemplate":null},{"cloudAccountRoleId":"d"}],"totalCount":4}`)
		}
	}))
	defer srv.Close()
	got, err := DiscoverAssumableRoles(srv.Client(), srv.URL, "instance", "test")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got.Entities) != 4 || got.TotalCount != 4 {
		t.Fatalf("incomplete result: %+v", got)
	}
	for i, want := range []string{"yes", "no", "unknown", "unknown"} {
		if v := TemplateCreationLabel(got.Entities[i].CreatedByPermissionTemplate); v != want {
			t.Errorf("label %d = %s", i, v)
		}
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]json.RawMessage
	if err := json.Unmarshal(b, &output); err != nil {
		t.Fatal(err)
	}
	if output["entities"] == nil || output["totalCount"] == nil || output["nextToken"] != nil {
		t.Fatalf("invalid aggregate JSON: %s", b)
	}
}

func TestDiscoveryRejectsFailuresWithoutPartialResults(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"http_error", `{"error":"OperationDenied.BatchSizeExceed"}`, 400},
		{"body_error", `{"error":"InvalidParameter.NextToken"}`, 200},
		{"signature_error", `{"code":"401","message":"signature failed"}`, 200},
		{"malformed", `broken`, 200},
		{"missing_entities", `{}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					fmt.Fprint(w, `{"entities":[{"cloudAccountRoleId":"a"}],"nextToken":"next"}`)
					return
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			got, err := DiscoverAssumableRoles(srv.Client(), srv.URL, "instance", "secret-token")
			if err == nil || got != nil {
				t.Fatal("must reject partial result")
			}
			if strings.Contains(err.Error(), "secret-token") {
				t.Fatal("token leaked")
			}
		})
	}
}

func TestDiscoveryRejectsCursorCycles(t *testing.T) {
	for _, cycle := range [][]string{{"A", "A"}, {"A", "B", "A"}} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls >= len(cycle) {
				t.Error("cycle not stopped")
				fmt.Fprint(w, `{"entities":[]}`)
				return
			}
			fmt.Fprintf(w, `{"entities":[],"nextToken":%q}`, cycle[calls])
			calls++
		}))
		got, err := DiscoverAssumableRoles(srv.Client(), srv.URL, "instance", "test")
		srv.Close()
		if err == nil || got != nil || calls != len(cycle) {
			t.Fatal("cursor cycle accepted")
		}
	}
}

func TestDiscoveryEmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"entities":[],"totalCount":0}`) }))
	defer srv.Close()
	got, err := DiscoverAssumableRoles(srv.Client(), srv.URL, "instance", "test")
	if err != nil || got == nil || got.Entities == nil || got.TotalCount != 0 {
		t.Fatalf("invalid empty result: %v", err)
	}
}
