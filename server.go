package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // подключенные клиенты
var broadcast = make(chan Message)           // канал сообщений

var upgrader = websocket.Upgrader{} // use default options

type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	fs := http.FileServer(http.Dir("../spws"))
	http.Handle("/", fs)
	// Маршрут для вебсокета
	http.HandleFunc("/ws", handleConnections)
	// Запуск горутины по прослушке входящих сообщений
	go handleMessages()

	log.Println("http server started on :8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Изменяем (upgade) первоначальный запрос GET на полный на WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Удаляем соединение при окончании горутины по какой-либо причине
	defer ws.Close()
	// Добавляем нового клиента
	clients[ws] = true

	for {
		var msg Message
		// Чтение и десериализация JSON в объект Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Отправка сообщений в канал
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Прием сообщений из канала
		msg := <-broadcast
		// Отправка принятого сообщения всем действительно подключенным клиентам
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
