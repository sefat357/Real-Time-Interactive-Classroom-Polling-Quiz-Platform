package main

import (
	"fmt"
	"log"

	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/google/uuid"
	_ "github.com/gorilla/websocket"
	_ "github.com/jung-kurt/gofpdf"
	_ "github.com/mattn/go-sqlite3"
	_ "golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("Real-Time Interactive Classroom Polling & Quiz Platform Server")
	log.Println("Core dependencies initialized successfully.")
}
