package main

import "log"

func main() {
	app := InitializeApp()
	if err := app.Service.Echo.StartTLS(app.config.Port, "./cert.pem", "./key.pem"); err != nil {
		log.Fatal(err)
		return
	}
}
