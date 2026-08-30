package bcrypt

import xbcrypt "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	hash, err := xbcrypt.GenerateFromPassword([]byte(password), xbcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hashed, password string) error {
	return xbcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}
