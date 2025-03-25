package main
import _ "github.com/lib/pq"
import (

	"log"
	"net/http"
	"fmt"
	"sync/atomic"
	"encoding/json"	
	"strings"
	"os"
	"database/sql"
	"github.com/joho/godotenv"
	"Chirpy/internal/database"
	"github.com/google/uuid"
	"time"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries  *database.Queries
	
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body     string    `json:"body"`
	User_id     uuid.UUID    `json:"user_id"`
}

func main (){

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, erru := sql.Open("postgres", dbURL)
	if erru != nil {
		log.Fatal(erru)
	} 
	dbQueries := database.New(db)

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

	apiCfg := apiConfig{
		queries : dbQueries,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app",fileserver)))
	mux.HandleFunc("GET /api/healthz", serverStatus)
	mux.HandleFunc("GET /admin/metrics", apiCfg.serverCount)
	mux.HandleFunc("POST /admin/reset", func(res http.ResponseWriter, req *http.Request) {
		apiCfg.resetServerCount(res, req)
	})
	mux.HandleFunc("GET /api/chirps", apiCfg.chirpsQueryAll)
	mux.HandleFunc("POST /api/chirps", apiCfg.chirpsCreator)
	mux.HandleFunc("POST /api/users", apiCfg.userCreator)

	
	
	//http.Server definisce una configurazione server
	server := http.Server{
		Handler : mux,
		Addr : ":8080",
	}

	log.Println("Starting server on :8080")
	err := server.ListenAndServe()

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
	plat := os.Getenv("PLATFORM")
	if plat != "dev" {
		log.Printf("non autorizzato %v", plat)
		res.WriteHeader(403)
		return
	}

	err := cfg.queries.DeleteUsers(req.Context())
	if err != nil {
		log.Printf("errore in cancellazione::: %v", err)
	}
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

func(cfg *apiConfig) chirpsQueryAll(res http.ResponseWriter, req *http.Request){

	type returnError struct{
		Error string `json:"error"`
	}

	var chirps []database.Chirp
	var outChirps []Chirp
	//crea il chirp
	chirps, err  := cfg.queries.QueryAllChirps(req.Context())
	if err != nil {
		log.Printf("errore in creazione::: %v", err)
	}
	log.Printf("chirps trovati::: %v", chirps)
	for _, c := range chirps {
		outputChirp := Chirp{
			ID : c.ID,
			CreatedAt : c.CreatedAt,
			UpdatedAt : c.UpdatedAt,
			Body : c.Body,
			User_id : c.UserID,
		}
		outChirps = append(outChirps, outputChirp)
	}


	data, err := json.Marshal(outChirps)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)

}

func (cfg *apiConfig) chirpsCreator (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Body string `json:"body"`
		User_id uuid.UUID `json:"user_id"`
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

	clearedParameters := database.CreateChirpParams{
		Body : clearingString,
		UserID : params.User_id,
	}
	var chirp database.Chirp
	//crea il chirp
	chirp, err = cfg.queries.CreateChirp(req.Context(), clearedParameters)
	if err != nil {
		log.Printf("errore in creazione::: %v", err)
	}
	log.Printf("body ricevuto::: %v", params.Body)
	log.Printf("chirp creato::: %v", chirp)
	outputChirp := Chirp{
		ID : chirp.ID,
		CreatedAt : chirp.CreatedAt,
		UpdatedAt : chirp.UpdatedAt,
		Body : chirp.Body,
		User_id : chirp.UserID,
	}

	data, err := json.Marshal(outputChirp)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(201)
	res.Write(data)

}

func (cfg *apiConfig) userCreator (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Email string `json:"email"`
	}

	type returnError struct{
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	//gestione errore
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

	var user database.User
	//crea l'utenza
	user, err = cfg.queries.CreateUser(req.Context(), params.Email)
	if err != nil {
		log.Printf("errore in creazione::: %v", err)
	}
	log.Printf("email ricevuta::: %v", params.Email)
	log.Printf("utenza creata::: %v", user)
	outputUser := User{
		ID : user.ID,
		CreatedAt : user.CreatedAt,
		UpdatedAt : user.UpdatedAt,
		Email : user.Email,
	}

	data, err := json.Marshal(outputUser)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(201)
	res.Write(data)
}
