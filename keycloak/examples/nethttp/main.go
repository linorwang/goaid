package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/linorwang/goaid/keycloak"
)

func main() {
	kc, err := keycloak.New(context.Background(), keycloak.Config{
		BaseURL:      "https://keycloak.example.com",
		Realm:        "master",
		ClientID:     "ops-platform",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/callback",
	})
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, kc.AuthCodeURL("replace-with-random-state"), http.StatusFound)
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := kc.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(token)
	})

	http.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		token, err := keycloak.BearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := kc.VerifyAccessToken(r.Context(), token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(claims)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
