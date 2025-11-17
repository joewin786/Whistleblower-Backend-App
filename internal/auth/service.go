package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var refreshJwtSecret = func() []byte {
	if s := os.Getenv("REFRESH_JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return jwtSecret
}()

func HashFunction(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(id string, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   id,
		"role": role,
		"typ":  "access",
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		typ, _ := claims["typ"].(string)
		if typ != "access" {
			return "", "", errors.New("invalid token type")
		}
		id, _ := claims["id"].(string)
		role, _ := claims["role"].(string)
		if id == "" || role == "" {
			return "", "", errors.New("invalid token claims")
		}
		return id, role, nil
	}
	return "", "", errors.New("invalid token claims")
}



func ValidateResetToken(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})
	
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}
	
	// ✅ Check typ = "reset"
	typ, _ := claims["typ"].(string)
	if typ != "reset" {
		return "", "", errors.New("invalid token type")
	}
	
	// ✅ Extract email (bukan id)
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)
	
	if email == "" || role == "" {
		return "", "", errors.New("invalid token claims")
	}
	
	// ✅ Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return "", "", errors.New("token expired")
		}
	}
	
	return email, role, nil
}

func GenerateRefreshToken(id string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  id,
		"typ": "refresh",
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	return token.SignedString(refreshJwtSecret)
}

func ValidateRefreshToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return refreshJwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		typ, _ := claims["typ"].(string)
		if typ != "refresh" {
			return "", errors.New("invalid token type")
		}
		if id, ok := claims["id"].(string); ok {
			return id, nil
		}
	}
	return "", errors.New("invalid token claims")
}

func ValidateChangePasswordToken(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})
	
	if err != nil || !token.Valid {
		return "", "", errors.New("invalid token")
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}
	
	// ✅ Check typ = "change_password"
	typ, _ := claims["typ"].(string)
	if typ != "change_password" {
		return "", "", errors.New("invalid token type")
	}
	
	email, _ := claims["email"].(string)
	role, _ := claims["role"].(string)
	
	if email == "" || role == "" {
		return "", "", errors.New("invalid token claims")
	}
	
	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return "", "", errors.New("token expired")
		}
	}
	
	return email, role, nil
}



func GenerateResetToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"role":  "reset",
		"typ":   "reset",  // ✅ typ = reset
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	})
	return token.SignedString(jwtSecret)
}
func GenerateChangePasswordToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"role":  "change_password",
		"typ":   "change_password",  // ✅ typ = change_password
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	})
	return token.SignedString(jwtSecret)
}