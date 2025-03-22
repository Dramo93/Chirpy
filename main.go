package main

import (

	"log"
	"net/http"
)

func main (){


	mux := http.NewServeMux()

	fileSystem := http.Dir(".")
	fileserver := http.FileServer(fileSystem)

	mux.Handle("/", fileserver)

	
	

	server := http.Server{
		Handler : mux,
		Addr : ":8080",
	}

	log.Println("Starting server on :8080")
	err:= server.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}
}