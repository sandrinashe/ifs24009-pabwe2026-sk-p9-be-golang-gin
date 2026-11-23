// Titik masuk di root agar `go build -o server .` (Jenkins) menghasilkan biner server.
package main

import (
	"log"

	"app/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
