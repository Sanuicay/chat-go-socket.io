package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
	"github.com/zishang520/socket.io/v2/socket"
)

var db *sql.DB

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Connection struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionId"`
}

var connectedUsers = make(map[string]string)

func main() {
	var err error
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/messaging_app")
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
		handshake := client.Handshake()
		sessionID := handshake.Query["sessionID"]
		name := handshake.Query["name"]

		if len(sessionID) == 0 || len(name) == 0 {
			log.Println("Missing sessionID or name")
			client.Disconnect(true)
			return
		}

		connectedUsers[sessionID[0]] = name[0]
		log.Printf("User connected: %s (SessionID: %s)\n", name[0], sessionID[0])
		addorUpdateUser(name[0])

		// List all online users
		onlineUsers := listOnlineUsers()
		log.Printf("Online users: %s\n", onlineUsers)

		client.On("ShowUsersChat", func(...any) {
			rows, err := db.Query("SELECT chat.id, chat.name FROM chat JOIN chat_users ON chat.id = chat_users.chat_id WHERE chat_users.user_id = (SELECT id FROM users WHERE name = ?)", name[0])
			if err != nil {
				log.Println(err)
			}
			defer rows.Close()

			for rows.Next() {
				var chatID int
				var chatName string
				err := rows.Scan(&chatID, &chatName)
				if err != nil {
					log.Println(err)
				}
				client.Emit("ShowUsersChatReply", chatID, chatName)
			}
		})

		client.On("ShowMessages", func(chatID ...any) {
			rows, err := db.Query("SELECT messages.message, users.name AS sender_name, messages.created_at FROM messages JOIN users ON messages.sender_id = users.id WHERE messages.chat_id = ?;", chatID[0])
			if err != nil {
				log.Println(err)
			}
			defer rows.Close()

			for rows.Next() {
				var message string
				var sender string
				var createdAt string
				err := rows.Scan(&message, &sender, &createdAt)
				if err != nil {
					log.Println(err)
				}
				client.Emit("ShowMessagesReply", message, sender, createdAt)
			}
		})

		// socket.emit('SendMessage', name, chatid, message);
		client.On("SendMessage", func(args ...interface{}) {
			name := args[0].(string)
			chatID := args[1].(string)
			message := args[2].(string)

			_, err := db.Exec("INSERT INTO messages (chat_id, sender_id, message) VALUES (?, (SELECT id FROM users WHERE name = ?), ?)", chatID, name, message)
			if err != nil {
				log.Println(err)
				client.Emit("SendMessageReply", "Failed to send message")
			} else {
				client.Emit("SendMessageReply", "Message sent")
			}
		})

		client.On("disconnect", func(...any) {
			log.Printf("User disconnected: %s (SessionID: %s)\n", name[0], sessionID[0])
			updateUserStatus(name[0], false)
			delete(connectedUsers, sessionID[0])
			onlineUsers := listOnlineUsers()
			log.Printf("Online users: %s\n", onlineUsers)
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

// if not exist, add user to users table (is_online equal True), else update the user is_online value to True
func addorUpdateUser(name string) {
	var user User
	err := db.QueryRow("SELECT id, name FROM users WHERE name = ?", name).Scan(&user.ID, &user.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err := db.Exec("INSERT INTO users (name, is_online) VALUES (?, ?)", name, true)
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Println(err)
		}
	} else {
		updateUserStatus(name, true)
	}
}

// change the bool value is_online in users table
func updateUserStatus(name string, status bool) {
	_, err := db.Exec("UPDATE users SET is_online = ? WHERE name = ?", status, name)
	if err != nil {
		log.Println(err)
	}
}

// list all online users as a string "(3) user1, user2, user3"
func listOnlineUsers() string {
	var (
		onlineUsers      string
		onlineUsersCount int
	)
	onlineUsersCount = 0
	rows, err := db.Query("SELECT name FROM users WHERE is_online = ?", true)
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var user string
		onlineUsersCount++
		err := rows.Scan(&user)
		if err != nil {
			log.Println(err)
		}
		onlineUsers += "(" + strconv.Itoa(onlineUsersCount) + ") " + user + " "
	}
	return onlineUsers
}

// func setEveryoneOffline() {
// 	_, err := db.Exec("UPDATE users SET is_online = ?", false)
// 	if err != nil {
// 		log.Println(err)
// 	}
// }
