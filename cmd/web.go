package cmd

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"

	errorhandler "sorcia/error"
	"sorcia/handler"
	"sorcia/middleware"
	"sorcia/model"
	"sorcia/setting"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// RunWeb ...
func RunWeb(conf *setting.BaseStruct) {
	// Mux initiate
	m := mux.NewRouter()

	// Open postgres database
	db := conf.DBConn
	defer db.Close()

	model.CreateAccount(db)
	model.CreateRepo(db)

	m.Use(middleware.Middleware)

	// Web handlers
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		GetHome(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogin(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handler.PostLogin(w, r, db, conf.Version, decoder, conf.Paths.RepoPath)
	}).Methods("POST")
	m.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handler.GetLogout(w, r)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.GetCreateRepo(w, r, db, conf.Version)
	}).Methods("GET")
	m.HandleFunc("/create-repo", func(w http.ResponseWriter, r *http.Request) {
		handler.PostCreateRepo(w, r, db, decoder, conf.Paths.RepoPath)
	}).Methods("POST")
	m.HandleFunc("/r/{reponame}", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepo(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.HandleFunc("/r/{reponame}/tree", func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTree(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame}/tree/{[a-zA-Z0-9]+}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GetRepoTreePath(w, r, db, conf.Version, conf.Paths.RepoPath)
	}).Methods("GET")
	m.PathPrefix("/r/{reponame[\\d\\w-_\\.]+\\.git$}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.GitviaHTTP(w, r, db, conf.Paths.RepoPath)
	}).Methods("GET", "POST")

	staticDir := filepath.Join(conf.Paths.ProjectRoot, "public")
	staticFileHandler := http.StripPrefix("/public/", http.FileServer(http.Dir(staticDir)))
	// The "PathPrefix" method acts as a matcher, and matches all routes starting
	// with "/public/", instead of the absolute route itself
	m.PathPrefix("/public/").Handler(staticFileHandler).Methods("GET")

	http.Handle("/", m)

	allowedOrigins := []string{"*"}
	allowedMethods := []string{"GET", "POST"}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", conf.Server.HTTPPort), handlers.CORS(handlers.AllowedOrigins(allowedOrigins), handlers.AllowedMethods(allowedMethods))(m)))
}

// IndexPageResponse struct
type IndexPageResponse struct {
	IsHeaderLogin    bool
	HeaderActiveMenu string
	SorciaVersion    string
	Username         string
	Repos            *model.GetReposFromUserIDResponse
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, sorciaVersion string) {
	userPresent := w.Header().Get("user-present")

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := model.GetUserIDFromToken(db, token)
		username := model.GetUsernameFromToken(db, token)
		repos := model.GetReposFromUserID(db, userID)

		layoutPage := path.Join("./templates", "layout.tmpl")
		headerPage := path.Join("./templates", "header.tmpl")
		indexPage := path.Join("./templates", "index.tmpl")
		footerPage := path.Join("./templates", "footer.tmpl")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		errorhandler.CheckError(err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsHeaderLogin:    false,
			HeaderActiveMenu: "header__menu--dashboard",
			SorciaVersion:    sorciaVersion,
			Username:         username,
			Repos:            repos,
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
