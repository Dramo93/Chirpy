package auth

import "golang.org/x/crypto/bcrypt"
import "errors"

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