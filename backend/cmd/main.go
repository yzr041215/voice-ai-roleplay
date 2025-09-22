package main

import "log"

func main() {
	app := InitializeApp()
	if err := app.Service.Echo.Start(app.config.Port); err != nil {
		log.Fatal(err)
		return
	}
}
