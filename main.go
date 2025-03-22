package main

import (

	"log"
	"net/http"
	"fmt"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

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

	apiCfg := apiConfig{}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app",fileserver)))
	mux.HandleFunc("/healthz", serverStatus)
	mux.HandleFunc("/metrics", apiCfg.serverCount)
	mux.HandleFunc("/reset", func(res http.ResponseWriter, req *http.Request) {
		apiCfg.resetServerCount(res, req)
	})

	
	
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

func serverStatus(res http.ResponseWriter, req *http.Request){
	res.Header().Set("Content-type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        cfg.fileserverHits.Add(1) // Safely increment the counter
        next.ServeHTTP(res, req)  // Pass control to the next handler
    })
}

func (cfg *apiConfig) resetServerCount(res http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	res.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) serverCount(res http.ResponseWriter, req *http.Request){
	res.Header().Set("Content-type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
	msg := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	res.Write([]byte(msg))
}