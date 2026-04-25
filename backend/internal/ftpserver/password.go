package ftpserver

import "golang.org/x/crypto/bcrypt"

func checkPassword(hash, raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
