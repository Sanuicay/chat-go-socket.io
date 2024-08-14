package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
	socketio "github.com/zishang520/socket.io/v2/socket"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Connection struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionId"`
}

type Message struct {
	ChatID  string    `json:"chatId"`
	Sender  string    `json:"sender"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

var connectedUsers = make(map[string]string)

var (
	db          *sql.DB
	redisClient *redis.Client
	io          *socketio.Server
)

func main() {
	var err error

	// Connect to MySQL database
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

	// Create a new socket.io server
	io = socketio.NewServer(nil, nil)

	// Create a new CORS handler
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

	// Connect to Redis
	opt, err := redis.ParseURL("redis://my-redis:@127.0.0.1:6379/0")
	if err != nil {
		panic(err)
	}

	redisClient = redis.NewClient(opt)

	pong, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Redis connected", pong)
	}

	// Call the message queue process function
	go processMessageQueue()

	// Handle incoming connections
	io.On("connection", func(clients ...any) {
		client := clients[0].(*socketio.Socket)
		handshake := client.Handshake()
		sessionID := handshake.Query["sessionID"]

		if len(sessionID) == 0 {
			log.Println("Missing sessionID")
			client.Disconnect(true)
			return
		}

		name := handshake.Query["name"]

		if len(sessionID) == 0 || len(name) == 0 {
			log.Println("Missing sessionID or name")
			client.Disconnect(true)
			return
		}

		connectedUsers[sessionID[0]] = name[0]
		log.Printf("User connected: %s (SessionID: %s)\n", name[0], sessionID[0])
		addOrUpdateUser(name[0])

		// List all online users
		onlineUsers := listOnlineUsers()
		log.Printf("Online users: %s\n", onlineUsers)

		client.On("ShowUsersChat", func(args ...any) {
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

		client.On("JoinChatRoom", func(clients ...any) {
			name := clients[0].(string)
			// search for chatid that "name" is in
			rows, err := db.Query("SELECT chat.id, chat.name FROM chat JOIN chat_users ON chat.id = chat_users.chat_id WHERE chat_users.user_id = (SELECT id FROM users WHERE name = ?)", name)
			if err != nil {
				log.Println(err)
			}
			defer rows.Close()
			//join the chat rooms
			for rows.Next() {
				var chatID int
				var chatName string
				err := rows.Scan(&chatID, &chatName)
				if err != nil {
					log.Println(err)
				}
				client.Join(socketio.Room(strconv.Itoa(chatID)))
				log.Printf("User %s joined chat room %s\n", name, chatName)
			}
		})

		client.On("ShowMessages", func(chatID ...any) {
			rows, err := db.Query("SELECT messages.message, users.name AS sender_name, messages.created_at FROM messages JOIN users ON messages.sender_id = users.id WHERE messages.chat_id = ? ORDER BY messages.id;", chatID[0])
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

		client.On("SendMessage", func(args ...any) {
			name := args[0].(string)
			chatID := args[1].(string)
			message := args[2].(string)

			msg := Message{
				ChatID:  chatID,
				Sender:  name,
				Content: message,
				Time:    time.Now(),
			}

			msgJSON, _ := json.Marshal(msg)

			// Publish the message
			if _, err := redisClient.Publish(context.Background(), "message_queue", chatID+":"+string(msgJSON)).Result(); err != nil {
				log.Println("Error publishing message:", err)
				client.Emit("SendMessageReply", "Failed to send message")
				return
			}

			client.Emit("SendMessageReply", "Message sent")
		})

		// client.On("CreateChat", func(args ...any) {
		// 	chatName := args[0].(string)
		// 	is_group := args[1].(bool)

		client.On("disconnect", func(...any) {
			log.Printf("User disconnected: %s (SessionID: %s)\n", name[0], sessionID[0])
			updateUserStatus(name[0], false)
			delete(connectedUsers, sessionID[0])
			onlineUsers := listOnlineUsers()
			log.Printf("Online users: %s\n", onlineUsers)
		})
	})

	// Handle shutdown signals
	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1)
	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				setEveryoneOffline()
				close(exit)
				return
			}
		}
	}()

	<-exit
	io.Close(nil)
	os.Exit(0)
}

func addOrUpdateUser(name string) {
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

func updateUserStatus(name string, status bool) {
	_, err := db.Exec("UPDATE users SET is_online = ? WHERE name = ?", status, name)
	if err != nil {
		log.Println(err)
	}
}

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

func setEveryoneOffline() {
	_, err := db.Exec("UPDATE users SET is_online = ?", false)
	if err != nil {
		log.Println(err)
	}
}

func processMessageQueue() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pubsub := redisClient.Subscribe(ctx, "message_queue")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		// Check for empty messages
		if msg.Payload == "" {
			log.Println("Empty message received from queue")
			continue
		}

		// Split the message to extract the chat ID and JSON data
		chatID, msgJSON, found := strings.Cut(msg.Payload, ":")
		if !found {
			log.Println("Invalid message format in queue:", msg.Payload)
			continue
		}

		// Check for empty message JSON after extraction
		if msgJSON == "" {
			log.Println("Empty message JSON received from queue:", msg.Payload)
			continue
		}

		// Validate the JSON before unmarshalling
		if !json.Valid([]byte(msgJSON)) {
			log.Println("Invalid JSON message received from queue:", msg.Payload)
			continue
		}

		// Unmarshal message
		var msg Message
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		// Insert message into database
		_, err := db.Exec("INSERT INTO messages (chat_id, sender_id, message, created_at) VALUES (?, (SELECT id FROM users WHERE name = ?), ?, ?)", msg.ChatID, msg.Sender, msg.Content, msg.Time)
		if err != nil {
			log.Println("Error inserting message:", err)
			continue
		}

		notification := fmt.Sprintf("Message \"%s\" successfully processed", msg.Content)
		if _, err := redisClient.Publish(ctx, "message_processed", notification).Result(); err != nil {
			log.Println("Error publishing notification:", err)
		}

		// Emit the message to the relevant room
		io.To(socketio.Room(chatID)).Emit("NewMessage", msg.Sender, msg.Content, msg.Time.Format(time.DateTime))
	}
}
