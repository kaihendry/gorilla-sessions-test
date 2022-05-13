package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCount(t *testing.T) {

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		panic(err)
	}

	session, _ := store.Get(r, sessionName)
	var role = "admin"

	// we check for this role later in the table driven test
	session.Values["role"] = role

	log.Println("Setting up session", session.Values)

	err = session.Save(r, w)
	if err != nil {
		panic(err)
	}

	log.Println("cookie header", w.HeaderMap)
	// convert w.HeaderMap to *http.Cookie
	cookies := []*http.Cookie{}
	for _, v := range w.HeaderMap["Set-Cookie"] {
		cookies = append(cookies, &http.Cookie{
			Name:  strings.Split(v, "=")[0],
			Value: strings.Split(v, "=")[1] + "==",
		})
	}

	tests := []struct {
		name               string
		path               string
		cookie             *http.Cookie
		expectedStatusCode int
		expectedHeader     string
	}{
		{"no cookie", "/", nil, http.StatusUnauthorized, ""},
		{"role in session", "/", cookies[0], http.StatusOK, role},
	}

	s := newServer(true)
	ts := httptest.NewServer(s.router)
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := ts.URL + tt.path

			client := &http.Client{}
			r, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				panic(err)
			}

			if tt.cookie != nil {
				log.Println("adding cookie", tt.cookie)
				r.AddCookie(tt.cookie)
			}
			resp, err := client.Do(r)

			if resp.StatusCode != tt.expectedStatusCode {
				t.Fatalf("got %d, want %d", resp.StatusCode, tt.expectedStatusCode)
			}

			role := resp.Header.Get("X-Role")
			log.Println("role", role)
			if !strings.Contains(role, tt.expectedHeader) {
				t.Errorf("got %v, want %v", role, tt.expectedHeader)
			}
		})
	}
}
