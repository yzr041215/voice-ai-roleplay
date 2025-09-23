package main

import "demo/domain"

func main() {
	app := InitializeApp()
	if err := app.db.DB.AutoMigrate(domain.User{}); err != nil {
		panic(err)
	}
}
