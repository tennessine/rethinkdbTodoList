package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

var (
	router  *mux.Router
	session *r.Session
)

func NewServer(addr string) *http.Server {
	router = initRouting()
	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}

func initRouting() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/all", indexHandler)
	r.HandleFunc("/active", activeIndexHandler)
	r.HandleFunc("/completed", completedIndexHandler)
	r.HandleFunc("/new", newHandler)
	r.HandleFunc("/toggle/{id}", toggleHandler)
	r.HandleFunc("/delete/{id}", deleteHandler)
	r.HandleFunc("/clear", clearHandler)

	r.HandleFunc("/ws/all", newChangesHandler(allChanges))
	r.HandleFunc("/ws/active", newChangesHandler(activeChanges))
	r.HandleFunc("/ws/completed", newChangesHandler(completedChanges))

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	return r
}

func StartServer(server *http.Server) {
	log.Println("Starting server")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}

func indexHandler(w http.ResponseWriter, req *http.Request) {
	items := make([]TodoItem, 0)

	cursor, err := r.Table("items").
		OrderBy(r.Asc("Created")).
		Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = cursor.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "all",
	})
}

func activeIndexHandler(w http.ResponseWriter, req *http.Request) {
	items := make([]TodoItem, 0)

	query := r.Table("items").
		Filter(r.Row.Field("Status").Eq("active"))
	query = query.OrderBy(r.Asc("Created"))
	cursor, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = cursor.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "active",
	})
}

func completedIndexHandler(w http.ResponseWriter, req *http.Request) {
	items := make([]TodoItem, 0)

	query := r.Table("items").
		Filter(r.Row.Field("Status").Eq("complete"))
	query = query.OrderBy(r.Asc("Created"))
	cursor, err := query.Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = cursor.All(&items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "index", map[string]interface{}{
		"Items": items,
		"Route": "complete",
	})
}

func newHandler(w http.ResponseWriter, req *http.Request) {
	item := NewTodoItem(req.PostFormValue("text"))
	item.Created = time.Now()

	_, err := r.Table("items").
		Insert(item).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func toggleHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	cursor, err := r.Table("items").Get(id).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cursor.IsNil() {
		http.NotFound(w, req)
		return
	}

	_, err = r.Table("items").Get(id).Update(map[string]interface{}{
		"Status": r.Branch(
			r.Row.Field("Status").Eq("active"),
			"complete",
			"active"),
	}).RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, req, "/", http.StatusFound)
}

func deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	if id == "" {
		http.NotFound(w, req)
		return
	}

	cursor, err := r.Table("items").Get(id).Run(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cursor.IsNil() {
		http.NotFound(w, req)
		return
	}

	_, err = r.Table("items").Get(id).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusFound)
}

func clearHandler(w http.ResponseWriter, req *http.Request) {
	_, err := r.Table("items").Filter(r.Row.Field("Status").Eq("complete")).Delete().RunWrite(session)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusInternalServerError)
}

func newChangesHandler(fn func(chan interface{})) http.HandlerFunc {
	h := newHub()
	go h.run()

	fn(h.broadcast)
	return wsHandler(h)
}

func init() {
	var err error

	session, err = r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "todo",
		MaxOpen:  40,
	})
	if err != nil {
		log.Fatalln(err)
	}
}
