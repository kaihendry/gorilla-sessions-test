package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/sessions"

	"github.com/apex/gateway/v2"
	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
)

var sessionName = "foobar"
var sessionKey = "just-testing"
var store = sessions.NewCookieStore([]byte(sessionKey))

type server struct {
	router *http.ServeMux
}

func MyHandler(w http.ResponseWriter, r *http.Request) {
	// show cookies on request
	log.WithField("cookies", r.Cookies()).Info("request cookies")
	session, err := store.Get(r, sessionName)
	if err != nil {
		log.WithError(err).Error("error getting session")
	}

	log.WithField("MyHandler session", session).Info("session")
	log.WithField("MyHandler session values", session.Values).Info("session")
	if _, ok := session.Values["role"]; !ok {
		// return unauthorized
		log.Warnf("unauthorized request from %s", r.RemoteAddr)
		// 		t, err := template.New("foo").Parse(`<!DOCTYPE html>
		// <html>
		// <head>
		// <meta charset="utf-8" />
		// </head>
		// <body>
		// <form action="/setRole" method="post">
		// <input type="text" name="role" />
		// <input type="submit" value="Set Role" />
		// </body>
		// </html>`)
		// 		if err != nil {
		// 			panic(err)
		// 		}
		// 		err = t.Execute(w, nil)
		// 		if err != nil {
		// 			panic(err)
		// 		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-Role", session.Values["role"].(string))
}

func setRole(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)

	// parse post request for role
	role := r.FormValue("role")
	log.WithField("role", role).Info("setting role")
	if role == "" {
		http.Error(w, "No role provided", http.StatusBadRequest)
		return
	}
	session.Values["role"] = role

	// Save it before we write to the response/return from the handler.
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// redirect back
	http.Redirect(w, r, "/", http.StatusFound)
}

func newServer(local bool) *server {
	s := &server{router: &http.ServeMux{}}

	if local {
		log.SetHandler(text.Default)
		log.Info("local mode")
	} else {
		log.SetHandler(jsonhandler.Default)
		log.Info("cloud mode")
	}

	s.router.Handle("/", http.HandlerFunc(MyHandler))
	s.router.Handle("/setRole", http.HandlerFunc(setRole))

	return s
}

func main() {
	_, awsDetected := os.LookupEnv("AWS_LAMBDA_FUNCTION_NAME")
	log.WithField("awsDetected", awsDetected).Info("starting up")
	s := newServer(!awsDetected)

	var err error

	if awsDetected {
		log.Info("starting cloud server")
		err = gateway.ListenAndServe("", s.router)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), s.router)
	}
	log.WithError(err).Fatal("error listening")
}
