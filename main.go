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
	"Chirpy/internal/auth"
	"sort"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries  *database.Queries
	secretToken string
	apiKey string
	
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token	  string 	`json:"token"`
	RefreshToken string `json:"refresh_token"`
	Is_chirpy_red bool  `json:"is_chirpy_red"`
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
	secretTokenConfig := os.Getenv("SECRETTOKEN")
	apik := os.Getenv("POLKA_KEY")

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
		secretToken : secretTokenConfig,
		apiKey : apik,
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app",fileserver)))
	mux.HandleFunc("GET /api/healthz", serverStatus)
	mux.HandleFunc("GET /admin/metrics", apiCfg.serverCount)
	mux.HandleFunc("POST /admin/reset", func(res http.ResponseWriter, req *http.Request) {
		apiCfg.resetServerCount(res, req)
	})
	mux.HandleFunc("GET /api/chirps", apiCfg.chirpsQueryAll)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.chirpsQuery)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.chirpsDelete)
	
	mux.HandleFunc("POST /api/chirps", apiCfg.chirpsCreator)
	mux.HandleFunc("POST /api/users", apiCfg.userCreator)
	mux.HandleFunc("PUT /api/users", apiCfg.modifyUser)
	mux.HandleFunc("POST /api/login", apiCfg.userLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeToken)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.upgradeUser)

	
	
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

	author := req.URL.Query().Get("author_id")
	sorting := req.URL.Query().Get("sort")
	if author == "" {
		//crea il chirp
		chirps, _ = cfg.queries.QueryAllChirps(req.Context())

	} else {
		authorId, _ := uuid.Parse(author)
		chirps, _ = cfg.queries.QueryAllAuthorChirps(req.Context(), authorId)
	
	}
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
	if sorting == "asc"{
		sort.Slice(outChirps, func(i, j int) bool { return outChirps[i].CreatedAt.Before(outChirps[j].CreatedAt) })
	}
	if sorting == "desc" {
		sort.Slice(outChirps, func(i, j int) bool { return outChirps[i].CreatedAt.After(outChirps[j].CreatedAt) })
	}


	data, _ := json.Marshal(outChirps)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)

}

func(cfg *apiConfig) chirpsQuery(res http.ResponseWriter, req *http.Request){

	type returnError struct{
		Error string `json:"error"`
	}

	chirpIDString := req.PathValue("chirpID")
	chirpID, _ := uuid.Parse(chirpIDString)

	//cerca il chirp
	chirp, err  := cfg.queries.QueryChirp(req.Context(), chirpID)
	if err != nil {
		log.Printf("errore in creazione::: %v", err)
	}
	if chirp.Body == "" {
		res.WriteHeader(404)
		return
	}
	log.Printf("chirp trovato::: %v", chirp)
		outputChirp := Chirp{
			ID : chirp.ID,
			CreatedAt : chirp.CreatedAt,
			UpdatedAt : chirp.UpdatedAt,
			Body : chirp.Body,
			User_id : chirp.UserID,
	}


	data, err := json.Marshal(outputChirp)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)

}

func(cfg *apiConfig) chirpsDelete(res http.ResponseWriter, req *http.Request){

	type returnError struct{
		Error string `json:"error"`
	}

	reqBearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("errore nel GetBearerToken")
		res.WriteHeader(500)
		return
	}
	userFound, err := auth.ValidateJWT(reqBearer, cfg.secretToken)
	if err != nil {
		log.Printf("unauthorized with this token:: %v, this error present %v", reqBearer, err)
		res.WriteHeader(401)
		return
	}


	chirpIDString := req.PathValue("chirpID")
	chirpID, _ := uuid.Parse(chirpIDString)

	//cerca il chirp
	chirp, err  := cfg.queries.QueryChirp(req.Context(), chirpID)
	if err != nil {
		log.Printf("chirp da cancellare non trovato::: %v", err)
	}
	if chirp.Body == "" {
		res.WriteHeader(404)
		return
	}

	if chirp.UserID != userFound {
		res.WriteHeader(403)
		return
	}

	err = cfg.queries.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		log.Printf("errore in cancellazione::: %v", err)
		res.WriteHeader(404)
	}


	res.WriteHeader(204)


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
	reqBearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("errore nel GetBearerToken")
		res.WriteHeader(500)
		return
	}
	userFound, err := auth.ValidateJWT(reqBearer, cfg.secretToken)
	if err != nil {
		log.Printf("unauthorized with this token:: %v, this error present %v", reqBearer, err)
		res.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
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
		UserID : userFound,
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
		User_id : userFound,
	}

	data, err := json.Marshal(outputChirp)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(201)
	res.Write(data)

}

func (cfg *apiConfig) userCreator (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Password string `json:"password"`
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
	hashed, _ := auth.HashPassword(params.Password)
	userParam := database.CreateUserParams{
		Email : params.Email,
		HashedPassword : hashed,
	}
	user, err = cfg.queries.CreateUser(req.Context(), userParam)
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
		Is_chirpy_red : user.IsChirpyRed,
	}

	data, err := json.Marshal(outputUser)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(201)
	res.Write(data)
}

func (cfg *apiConfig) userLogin (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Password string `json:"password"`
		Email string `json:"email"`
		//ExpSec int `json:"expires_in_seconds"`
	}

	type returnError struct{
		Error string `json:"error"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	//gestione errore
	if err != nil {
		responseBody := returnError{
			Error : "errore",
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


	user, err  = cfg.queries.QueryUser(req.Context(), params.Email)
	if err != nil {
		log.Printf("errore in query::: %v", err)
		return
	}
	if user.Email == "" {
		res.WriteHeader(401)
		log.Printf("incorrect email or password")
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("incorrect email or password")
		return
	}
	outputUser := User{
		ID : user.ID,
		CreatedAt : user.CreatedAt,
		UpdatedAt : user.UpdatedAt,
		Email : user.Email,
		Is_chirpy_red : user.IsChirpyRed,
	}

	expiration := 3600
	//generate Access Token
	userToken, err := auth.MakeJWT(user.ID, cfg.secretToken, time.Duration(expiration)* time.Second)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("problem  with token:: %v", err)
		return
	}
	outputUser.Token = userToken

	//generate Refresh Token
	refreshToken, err := auth.MakeRefreshToken()
	outputUser.RefreshToken = refreshToken

	tm := time.Now()
	tm = tm.AddDate(0, 0, 60) 
	refTokenPar := database.CreateRefreshTokenParams{
		Token : refreshToken,
		UserID : user.ID,
		ExpiresAt : tm,
	}
	refreshTokenCreated, err := cfg.queries.CreateRefreshToken(req.Context(), refTokenPar)
	log.Printf("creatoRefreshToken:: %v", refreshTokenCreated)
	log.Printf("tempo di expire:: %v", tm)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("problem  with refreshToken:: %v", err)
		return
	}


	data, err := json.Marshal(outputUser)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)
}

func (cfg *apiConfig) refreshToken (res http.ResponseWriter, req *http.Request){

	type returnToken struct{
		Token string `json:"token"`
	}
	reftoken, err := auth.GetBearerToken(req.Header)
	foundToken, err := cfg.queries.QueryRefreshToken(req.Context(), reftoken)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("refresh token not found:: %v", err)
		return 
	}
	// Verifica se il token è stato revocato
	if foundToken.RevokedAt.Valid {
		res.WriteHeader(401)
		log.Printf("token revoked at:: %v", foundToken.RevokedAt.Time)
		return 
	}

	// Verifica separata per la scadenza
	if foundToken.ExpiresAt.Before(time.Now()) {
		res.WriteHeader(401)
		log.Printf("token expired:: %v", foundToken.ExpiresAt)
		return 
	}

	
	expiration := 3600
	//generate Access Token
	userToken, err := auth.MakeJWT(foundToken.UserID, cfg.secretToken, time.Duration(expiration)* time.Second)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("problem  with new access token:: %v", err)
		return 
	}
	outputToken := returnToken{
		Token :  userToken,
	}
	data, err := json.Marshal(outputToken)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	res.Write(data)

}

func (cfg *apiConfig) revokeToken (res http.ResponseWriter, req *http.Request){
	reftoken, _ := auth.GetBearerToken(req.Header)

	revoked, err := cfg.queries.RevokeToken(req.Context(), reftoken)
	log.Printf("revoked token::: %v ::: token completed", revoked)
	if err != nil {
		res.WriteHeader(401)
		log.Printf("problem  with token:: %v", err)
		return 
	}
	res.WriteHeader(204)
	return

}

func (cfg *apiConfig) modifyUser (res http.ResponseWriter, req *http.Request){
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	type returnError struct{
		Error string `json:"error"`
	}
	reqBearer, err := auth.GetBearerToken(req.Header)
	if err != nil {
		log.Printf("errore nel GetBearerToken")
		res.WriteHeader(401)
		return
	}
	userFound, err := auth.ValidateJWT(reqBearer, cfg.secretToken)
	if err != nil {
		log.Printf("unauthorized with this token:: %v, this error present %v", reqBearer, err)
		res.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
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

	hashed, _ := auth.HashPassword(params.Password)
	userParam := database.UpdateUserParams{
		ID : userFound,
		Email : params.Email,
		HashedPassword : hashed,
	}
	user, err := cfg.queries.UpdateUser(req.Context(), userParam)
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
	res.WriteHeader(200)
	res.Write(data)
}

func (cfg *apiConfig) upgradeUser (res http.ResponseWriter, req *http.Request){
	type dataParam struct {
		User_id string `json:"user_id"`
	}

	type webhookParams struct {
		Event string `json:"event"`
		Data dataParam `json:"data"`
	}

	type returnError struct{
		Error string `json:"error"`
	}

	apiKeyFound, err := auth.GetAPIKey(req.Header)
	if err != nil {
		log.Printf("errore riscontrato nel prendere la api key::: %v", err)
		res.WriteHeader(401)
		return
	}

	if apiKeyFound != cfg.apiKey {
		log.Printf("unauthorized with this apikey:: %v", apiKeyFound)
		res.WriteHeader(401)
		return
	}

	decoder := json.NewDecoder(req.Body)
	whParams := webhookParams{}
	err = decoder.Decode(&whParams)

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

	if whParams.Event != "user.upgraded" {
		res.WriteHeader(204)
		return
	}
	user, _ := uuid.Parse(whParams.Data.User_id)
	_, err = cfg.queries.UserPro(req.Context(), user)
	if err != nil {
		log.Printf("errore riscontrato::: %v", err)
		res.WriteHeader(404)
	}
	res.WriteHeader(204)
	return
}

