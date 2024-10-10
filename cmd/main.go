package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"strconv"
)

const (
	CODE_NEW_GAME        = 5
	CODE_CONNECT_TO_GANE = 10
	CODE_MOVE_IN_GAME    = 15
)

var gameList = make(map[int]*Game)

type Game struct {
	players [2]*websocket.Conn
	gameId  int
	area    Area
}

type Area struct {
	Field [3][3]Field
}

// Инициализация игрового поля
func (area *Area) init() {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			area.Field[i][j] = Field{
				area:      area,
				xPosition: i,
				yPosition: j,
			}
		}
	}
}

func (area *Area) findFieldByCoordinate(xPosition int, yPosition int) *Field {
	return &area.Field[xPosition][yPosition]
}

type Field struct {
	area      *Area
	value     string
	xPosition int
	yPosition int
}

func checkVerticalLine(area Area, xPosition int, yPosition int, point string) bool {
	chooseNeighborWin := 0
	for fPointer := xPosition; fPointer >= 0; fPointer-- {
		if area.Field[fPointer][yPosition].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}
	}

	for fPointer := xPosition + 1; fPointer < 3; fPointer++ {
		if area.Field[fPointer][yPosition].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}
	}

	return false
}

func checkHorizontalLine(area Area, xPosition int, yPosition int, point string) bool {
	chooseNeighborWin := 0
	for pointer := yPosition; pointer >= 0; pointer-- {
		if area.Field[xPosition][pointer].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true

			}
		}
	}

	for pointer := yPosition + 1; pointer < 3; pointer++ {
		if area.Field[xPosition][pointer].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}
	}

	return false
}

func checkDiagonals(area Area, xPosition int, yPosition int, point string) bool {
	chooseNeighborWin := 0
	// проверяем одну диагональ
	pointerX := xPosition
	pointerY := yPosition
	for pointerX >= 0 && pointerY >= 0 {
		if area.Field[pointerX][pointerY].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}

		pointerY--
		pointerX--
	}

	pointerX = xPosition + 1
	pointerY = yPosition + 1
	for pointerX < 3 && pointerY < 3 {
		if area.Field[pointerX][pointerY].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}

		pointerY++
		pointerX++
	}

	chooseNeighborWin = 0
	// Проверяем вторую диагональ
	pointerX = xPosition
	pointerY = yPosition
	for pointerX >= 0 && pointerY < 3 {
		if area.Field[pointerX][pointerY].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}

		pointerX--
		pointerY++
	}

	pointerX = xPosition + 1
	pointerY = yPosition - 1
	for pointerX < 3 && pointerY >= 0 {
		if area.Field[pointerX][pointerY].value == point {
			chooseNeighborWin++
			if chooseNeighborWin == 3 {
				return true
			}
		}

		pointerX++
		pointerY--
	}

	return false
}

func (game *Game) checkGameWin(positionX int, positionY int, point string) bool {
	if checkVerticalLine(game.area, positionX, positionY, point) {
		return true
	} else if checkHorizontalLine(game.area, positionX, positionY, point) {
		return true
	} else if checkDiagonals(game.area, positionX, positionY, point) {
		return true
	}

	return false
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Пропускаем любой запрос
	},
}

var clientWebsocket map[*websocket.Conn]bool

func simpleWebsocket(response http.ResponseWriter, request *http.Request) {
	connection, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		panic(err)
	}

	defer delete(clientWebsocket, connection)
	defer connection.Close()
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			break
		}

		for client := range clientWebsocket {
			client.WriteMessage(websocket.TextMessage, message)
		}

		connection.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Сообещние в ответ на %s", string(message))))
		go func(message []byte) {
			fmt.Println(fmt.Sprintf("Печать сообщения: %s", string(message)))
		}(message)
	}
}

func main() {
	http.HandleFunc("/simple-websocket", simpleWebsocket)
	http.HandleFunc("/new-network-game", createNetworkGame)
	http.HandleFunc("/connect-network-game", connectNetworkGame)

	http.ListenAndServe(":8085", nil)
}

// Создание игры
func createNetworkGame(response http.ResponseWriter, request *http.Request) {
	id := rand.Intn(10000000)
	area := Area{}
	area.init()

	game := &Game{
		gameId: id,
		area:   area,
	}

	gameList[id] = game
	response.Write([]byte(strconv.Itoa(id)))
}

// Подключение к игре игрока
func connectNetworkGame(response http.ResponseWriter, request *http.Request) {
	connection, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		panic(err)
	}

	defer connection.Close()
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil || messageType == websocket.CloseMessage {
			break
		}

		for client := range clientWebsocket {
			client.WriteMessage(websocket.TextMessage, message)
		}

		var serverData ServerData
		err = json.Unmarshal(message, &serverData)
		if err != nil {
			fmt.Println(err)
		}

		switch clientRequest := serverData.Content.(type) {
		case ConnectToGame:
			game := gameList[clientRequest.GameId]
			if game == nil {
				connection.WriteMessage(websocket.TextMessage, []byte("Игра не найдена"))
			} else {
				if game.players[0] == nil {
					game.players[0] = connection
				} else if game.players[1] == nil {
					game.players[1] = connection

					connect := ConnectToGame{GameId: game.gameId, PointMove: "X"}
					responseWebsocket := ServerData{Code: CODE_CONNECT_TO_GANE, Content: connect}

					preparedMessageForPlayer1, _ := json.Marshal(responseWebsocket)
					game.players[0].WriteMessage(websocket.TextMessage, preparedMessageForPlayer1)

					connect2 := ConnectToGame{GameId: game.gameId, PointMove: "0"}
					responseWebsocket2 := ServerData{Code: CODE_CONNECT_TO_GANE, Content: connect2}

					preparedMessageForPlayer2, _ := json.Marshal(responseWebsocket2)
					game.players[1].WriteMessage(websocket.TextMessage, preparedMessageForPlayer2)
				} else {
					connection.WriteMessage(websocket.TextMessage, []byte("В эту игру уже играют 2 игроков"))
				}
			}
		case RequestMoveInGame:
			game := gameList[clientRequest.GameId]

			field := game.area.findFieldByCoordinate(clientRequest.PositionX, clientRequest.PositionY)
			field.value = clientRequest.PointMove

			isWin := game.checkGameWin(clientRequest.PositionX, clientRequest.PositionY, clientRequest.PointMove)
			fmt.Println(isWin)
			responseMove1 := ResponseMoveInGame{
				GameId:    game.gameId,
				PointMove: clientRequest.PointMove,
				PositionX: clientRequest.PositionX,
				PositionY: clientRequest.PositionY,
				IsWin:     isWin,
			}

			responseWebsocket := ServerData{Code: CODE_MOVE_IN_GAME, Content: responseMove1}

			preparedMessageForPlayer1, _ := json.Marshal(responseWebsocket)
			game.players[0].WriteMessage(websocket.TextMessage, preparedMessageForPlayer1)

			responseMove2 := ResponseMoveInGame{
				GameId:    game.gameId,
				PointMove: clientRequest.PointMove,
				PositionX: clientRequest.PositionX,
				PositionY: clientRequest.PositionY,
				IsWin:     isWin,
			}

			responseWebsocket2 := ServerData{Code: CODE_MOVE_IN_GAME, Content: responseMove2}

			preparedMessageForPlayer2, _ := json.Marshal(responseWebsocket2)
			game.players[1].WriteMessage(websocket.TextMessage, preparedMessageForPlayer2)
		default:
			fmt.Println("Не удалось распознать запрос клиента")
		}
	}
}

type ServerData struct {
	Code    int
	Content interface{}
}

type ConnectToGame struct {
	GameId    int
	PointMove string
}

type RequestMoveInGame struct {
	GameId    int
	PointMove string
	PositionX int
	PositionY int
}

type ResponseMoveInGame struct {
	GameId    int
	PointMove string
	PositionX int
	PositionY int
	IsWin     bool
}

func (sr *ServerData) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err := json.Unmarshal(raw["Code"], &sr.Code); err != nil {
		return err
	}

	switch sr.Code {
	case CODE_MOVE_IN_GAME:
		var content RequestMoveInGame
		if err := json.Unmarshal(raw["Content"], &content); err != nil {
			return err
		}
		sr.Content = content
	case CODE_CONNECT_TO_GANE:
		var content ConnectToGame
		if err := json.Unmarshal(raw["Content"], &content); err != nil {
			return err
		}
		sr.Content = content
	}

	return nil
}
