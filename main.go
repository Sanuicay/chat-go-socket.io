package main

import (
	"database/sql"
	// "time"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
	"github.com/zishang520/socket.io/v2/socket"
)

var db *sql.DB

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	var err error
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/messaging_app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Database connected")
	}

	io := socket.NewServer(nil, nil)

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
	})

	http.Handle("/socket.io/", io.ServeHandler(nil))
	http.Handle("/", http.FileServer(http.Dir("./public")))

	go func() {
		if err := http.ListenAndServe(":3000", c.Handler(http.DefaultServeMux)); err != nil {
			log.Fatal(err)
		}
	}()

	io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		log.Println("Client connected")

		client.On("message", func(datas ...any) {
			msg := datas[0].(string)
			log.Println("Received message:", msg)
			client.Emit("reply", "received "+msg)

			//save the message to the message table
			_, err = db.Exec("INSERT INTO messages (user_id, message) VALUES (?, ?)", 1, msg)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Message saved:", msg)
			}
		})

		client.On("register", func(datas ...any) {
			dataMap := datas[0].(map[string]interface{})
			username := dataMap["username"].(string)
			password := dataMap["password"].(string)
			log.Println("Received register:", username, password)

			//save the user to the user table
			_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
			if err != nil {
				log.Println(err)
				client.Emit("registerReply", "Error saving user: "+username)
			} else {
				log.Println("User saved:", username, password)
				client.Emit("registerReply", "User saved: "+username)
			}
		})

		client.On("login", func(datas ...any) {
			dataMap := datas[0].(map[string]interface{})
			username := dataMap["username"].(string)
			password := dataMap["password"].(string)
			log.Println("Received login:", username, password)
			var user User
			err = db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&user.Password)
			if err != nil {
				log.Println(err)
				client.Emit("loginReply", "User not found: "+username)
			} else {
				log.Println("User found:", username)
				if user.Password == password {
					log.Println("User logged in:", username)
					client.Emit("loginReplySuccess", "User logged in: "+username)
				} else {
					log.Println("User not logged in:", username)
					client.Emit("loginReplyFail", "User not logged in: "+username)
				}
			}
		})

		client.On("getMessages", func(datas ...any) {
			rows, err := db.Query("SELECT message FROM messages")
			if err != nil {
				log.Println(err)
			} else {
				for rows.Next() {
					var message string
					err = rows.Scan(&message)
					if err != nil {
						log.Println(err)
					} else {
						log.Println("Message:", message)
						client.Emit("message", message)
					}
				}
			}
		})
		client.On("disconnect", func(...any) {
			log.Println("Client disconnected")
		})
	})

	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1)
	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	io.Close(nil)
	os.Exit(0)
}
