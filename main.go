package main

import (

	"log"
	"net/http"
)

func main (){


	mux := http.NewServeMux()

	/*	The .Handle() method is how you register a handler function for a specific URL path in your server. In this case, you need to register a handler for the root path (/), which is what browsers request when someone visits your base URL (http://localhost:8080).

	You're not actually setting up a special handler for index.html specifically. Instead, what's happening is:

	You're setting up a FileServer that points to your current directory (.)
	When someone visits the root path (/), the FileServer automatically looks for an index.html file in the directory
	This is a standard convention in web servers - when a directory is requested, the server looks for an index.html file to serve
	So when you use:

	mux.Handle("/", http.FileServer(http.Dir(".")))

	You're telling the server "when someone requests '/', serve files from the current directory" - and the FileServer automatically knows to serve index.html when the root of that directory is requested.

	*/
	fileSystem := http.Dir(".")
	fileserver := http.FileServer(fileSystem)

	mux.Handle("/", fileserver)

	
	
	//http.Server definisce una configurazione server
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