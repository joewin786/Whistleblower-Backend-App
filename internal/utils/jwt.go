package utils

import (
    "os"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID, role string) (string, error) {
    secret := os.Getenv("JWT_SECRET") // ambil dari .env

    claims := jwt.MapClaims{
        "user_id": userID,
        "role":    role,
        "exp":     time.Now().Add(time.Hour * 72).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
