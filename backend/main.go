package main

import (
	"log"

	"platnova/backend/api"

	"github.com/gin-gonic/gin"
)

func main() {
	seedDemoData()

	r := gin.Default()
	api.RegisterRoutes(r)

	log.Println("Platnova wallet API listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// seedDemoData gives the dashboard something to show on first load.
func seedDemoData() {
	type walletSeed struct {
		currency string
		name     string
		deposit  float64
	}
	demoUsers := []struct {
		name    string
		email   string
		wallets []walletSeed
	}{
		{
			name:  "Daniel Effiong",
			email: "danieleffex@gmail.com",
			wallets: []walletSeed{
				{"USD", "USD Wallet", 4200.50},
				{"NGN", "Naira Wallet", 750000},
			},
		},
		{
			name:  "Amara Chukwu",
			email: "amara.chukwu@example.com",
			wallets: []walletSeed{
				{"NGN", "Naira Wallet", 1250000},
				{"EUR", "Euro Wallet", 900},
			},
		},
		{
			name:  "James Whitfield",
			email: "james.whitfield@example.com",
			wallets: []walletSeed{
				{"GBP", "GBP Wallet", 3200},
				{"USD", "USD Wallet", 1500},
			},
		},
	}

	for _, du := range demoUsers {
		user, err := api.CreateUser(du.name, du.email)
		if err != nil {
			log.Fatalf("failed to seed user %s: %v", du.name, err)
		}
		for _, w := range du.wallets {
			acc, err := api.CreateAccount(user.ID, w.name, w.currency)
			if err != nil {
				log.Fatalf("failed to seed wallet %s for %s: %v", w.name, du.name, err)
			}
			if _, _, err := api.Deposit(acc.ID, w.deposit); err != nil {
				log.Fatalf("failed to seed deposit for %s: %v", w.name, err)
			}
		}
	}
}
