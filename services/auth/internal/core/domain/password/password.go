package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

func HashPassword(pass string) (string, error){
	const (
		time = 3
		memory = 64 * 1024
		threads = 4
		keyLen = 32
		saltLen = 16
	)
	salt := make([]byte, saltLen)

	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	hash := computeHash(pass, salt, time,memory,threads,keyLen)
	
	salt64 := base64.RawStdEncoding.EncodeToString(salt)
	hash64 := base64.RawStdEncoding.EncodeToString(hash)

	result := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, salt64, hash64)
	return result, nil
}

func VerifyPassword(pass, encodedHash string) (bool, error) {

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}


	if parts[1] != "argon2id" {
		return false, errors.New("unsupported algorithm")
	}
	if parts[2] != "v=19" {
		return false, errors.New("unsupported version")
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errors.New("invalid parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid salt encoding")
	}
	oldHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errors.New("invalid hash encoding")
	}

	newHash := computeHash(pass, salt, time, memory, threads, uint32(len(oldHash)))

	if subtle.ConstantTimeCompare(newHash, oldHash) == 1 {
		return true, nil
	}
	return false, nil
}

func computeHash(pass string, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
    return argon2.IDKey([]byte(pass), salt, time, memory, threads, keyLen)
}

