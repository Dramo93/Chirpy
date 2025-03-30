package auth

import "golang.org/x/crypto/bcrypt"
import "errors"
import "github.com/google/uuid"
import "time"
import "github.com/golang-jwt/jwt/v5"
import "fmt"

type auth struct {}

func HashPassword(password string) (string, error){
	hashedPsw, err := bcrypt.GenerateFromPassword([]byte(password),16)
	if err != nil {
		return "", errors.New("errore nell'hashing")
	}
	return string(hashedPsw), nil
}

func CheckPasswordHash(hash, password string) error{
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return errors.New("errore nell'hashing")
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){

	/*type MyCustomClaims struct {
		Foo string `json:"foo"`
		jwt.RegisteredClaims
	}
	*/

	// Create claims while leaving out some of the optional fields
	//claims = MyCustomClaims{
	//	"bar",
	claims :=	jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Issuer:    "chirpy",
			Subject: userID.String(),
		}
	//}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(tokenSecret))
	fmt.Println(ss, err)
	return ss, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, errors.New("error during parsing")
	} else if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		fmt.Println(claims.Issuer)
		if token.Valid {
			userID, _ := uuid.Parse(claims.Subject)
			return userID, nil
		} else {
			return uuid.Nil, errors.New("invalid token")
		}

	} else {
		return uuid.Nil, errors.New("unknown claims type, cannot proceed")
	}

}