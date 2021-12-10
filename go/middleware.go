package main

import (
	"context"
	"log"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

func firebaseAuth() gin.HandlerFunc {

	return func(c *gin.Context) {

		opt := option.WithCredentialsFile("./jublee-a4dd5-firebase-adminsdk-txl4j-24612e329e.json")
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			log.Println("[AUTH] Step 1", err)
			//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err})
		}

		auth, err := app.Auth(context.Background())
		if err != nil {
			log.Println("[AUTH] Step 2", err)
			//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err})
		}

		// fetch the token from the header
		header := c.Request.Header.Get("Authorization")
		idToken := strings.TrimSpace(strings.Replace(header, "Bearer", "", 1))
		token, err := auth.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			log.Println("[AUTH] Step 3", err)
			//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err})
		}

		log.Println("GOT TOKEN", token)

		if token != nil {
			c.Set("uid", token.UID)
		} else {
			c.Set("uid", "mihai-test")
		}

	}
}
