package main

import (

	"log"
	"net/http"
	"fmt"
	"sync/atomic"
	"encoding/json"	
	"strings"
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
	mux.HandleFunc("GET /api/healthz", serverStatus)
	mux.HandleFunc("GET /admin/metrics", apiCfg.serverCount)
	mux.HandleFunc("POST /admin/reset", func(res http.ResponseWriter, req *http.Request) {
		apiCfg.resetServerCount(res, req)
	})
	mux.HandleFunc("POST /api/validate_chirp", chirpsValidator)

	
	
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
	res.Header().Set("Content-type", "text/html")
	res.WriteHeader(http.StatusOK)
	msg := fmt.Sprintf(`<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
		</html>`, cfg.fileserverHits.Load())
	res.Write([]byte(msg))
}

func chirpsValidator (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct{
		Clean string `json:"cleaned_body"`
	}
	type returnError struct{
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		err := "Something went wrong"

		responseBody := returnError{
			Error : err,
		}
		data, e := json.Marshal(responseBody)
		if e != nil {
			log.Printf("errore nel marshaling")
			res.WriteHeader(500)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(500)
		res.Write(data)
		return
	}

	if len(params.Body) > 140 {
		err := "Chirp is too long"

		responseBody := returnError{
			Error : err,
		}
		data, e := json.Marshal(responseBody)
		if e != nil {
			log.Printf("errore nel marshaling")
			res.WriteHeader(500)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(400)
		res.Write(data)
		return
	}

	clearingString := params.Body
	clearingString = strings.Replace(clearingString," kerfuffle " , " **** ", -1)
	clearingString = strings.Replace(clearingString," Kerfuffle " , " **** ", -1)
	clearingString = strings.Replace(clearingString," sharbert " , " **** ", -1)
	clearingString = strings.Replace(clearingString, "Sharbert ", " **** ", -1)
	clearingString = strings.Replace(clearingString, " fornax ", " **** ", -1)
	clearingString = strings.Replace(clearingString, " Fornax ", " **** ", -1)


	responseBody := returnVals{
		Clean: clearingString,
	}
	data, err := json.Marshal(responseBody)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)

}