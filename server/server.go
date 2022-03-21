package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func gethandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func main() {

	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path("/get").HandlerFunc(gethandler)

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.Handle("/", router)
	fmt.Println("Starting Server")
	//err := http.ListenAndServe(":8080", nil)

	var server *http.Server
	server = &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	err := server.ListenAndServeTLS("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}
}
