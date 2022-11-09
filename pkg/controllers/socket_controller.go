package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/olahol/melody"
	"github.com/sk25469/scribble_backend/pkg/model"
	"github.com/sk25469/scribble_backend/pkg/utils"
)

//	TODO: Logic for setting co-ordinates to draw, for every room we will store the co-ordinates which have already been drawn on the screen

var (

	//	Router for the web-socket
	// mrouter = config.GetWebSocketRouter()

	//	Response given by the server (can be changed according to the kind of request we want to change)
	response model.ServerResponse

	//	ID -> Room (private)
	privateRoomsMap map[string]*model.Room = make(map[string]*model.Room)

	//	ID -> Room (private)
	publicRoomsMap map[string]*model.Room = make(map[string]*model.Room)

	// 1st key is roomID, 2nd key is clientID and value is session
	totalClientsInSession map[string](map[string]*melody.Session) = make(map[string]map[string]*melody.Session)

	// zap logger
	// logger = config.GetLogger()

	//	Room Buckets
	publicRoomBucket *utils.RoomBucket = utils.Init()

	//	co-ordinates in a room
	drawnPoints map[string][]model.Point = make(map[string][]model.Point)
)

// TYPES OF REQUEST SENT BY SERVER
//
//  1. "iam" : The newly joined client sends a name and the kind of room it wants to join
//
//  2. "total" : Other clients are informed if any new client joins that room
//
//  3. "set" : When a client is drawing, it will send its x,y co-ordinate to others in the room
//
//  4. "dis" : Informs others in the room that "id" has disconnected
func OnConnect(s *melody.Session) {

	fmt.Printf("new client joined\n")

	response = model.ServerResponse{ResponseType: "iam", ClientInfo: model.ClientInfo{}, RoomInfo: model.Room{}, Point: model.Point{}}
	jsonResponse, err := json.Marshal(&response)
	if err != nil {
		log.Print("can't marshall reponse")
	}

	err = s.Write([]byte(jsonResponse))
	fmt.Printf("Written to the client: %v\n", response)

	if err != nil {
		log.Printf("Error in connection: %v\n", err)
		return

	}

}

// Will be triggered when the client "s" disconnects
func OnDisconnect(s *melody.Session) {
	info := s.MustGet("info").(*model.ClientInfo)
	roomID := info.RoomID
	if _, ok := privateRoomsMap[roomID]; ok {
		UpdateEverythingAfterDisconnect(*info, roomID, "private")
	} else {
		UpdateEverythingAfterDisconnect(*info, roomID, "public")
	}

}

// TYPES OF REQUEST SENT BY CLIENT
//
//  1. "connect-new" : When a client has entered its name and he wants to join a new room
//
//  2. "connect" : Client wants to connect to an existing room with ID
//
//  3. "move" : A client is drawing on the screen
//     "new" : A new line has been started
//     "update": The current line is being updated
//     "end" : The line has been drawn and mouse has been unclicked
//
//  4. "clear" : Client wants to clear the canvas
func OnMessage(s *melody.Session, msg []byte) {
	var clientResponse *model.ClientResponse

	//	take the response from the client and convert it to json
	err := json.Unmarshal(msg, &clientResponse)
	if err != nil {
		log.Printf("error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		log.Printf("response: %v", clientResponse)
		return
	}
	log.Printf("Client response %v", clientResponse)
	// create a new id for the client here
	id := uuid.NewString()
	clientID := id
	clientName := clientResponse.ClientInfo.Name

	// set up the new client info with the name we got from the client
	newClientInfo := model.ClientInfo{ClientID: clientID, Name: clientName}
	log.Printf("New client info: %v", newClientInfo)
	if clientResponse.ReponseType == "connect-new" {
		log.Printf("User wants to connect-new")
		var newRoomID string

		//	if he wants to create a private room, a new key for room is created
		if clientResponse.RoomType == "private" {
			newRoomID = utils.GetKey()
			newClientInfo.RoomID = newRoomID

			newRoom, err := AddAndUpdatePublicRooms([]model.ClientInfo{}, []model.ClientInfo{}, newClientInfo, newRoomID)
			if err != nil {
				log.Printf("Max no. of clients reached")
				return
			}
			privateRoomsMap[newRoomID] = newRoom
			log.Printf("User has been assigned %v and put in privateRoomsMap", newRoomID)
			jsonResponse, err := json.Marshal(model.ServerResponse{ResponseType: "total", ClientInfo: model.ClientInfo{RoomID: newRoomID, ClientID: clientID, Name: clientName}, RoomInfo: *newRoom, Point: model.Point{}})
			if err != nil {
				log.Printf("error parsing json")
				return
			}
			s.Write([]byte(jsonResponse))
			log.Printf("Sent client %v", string(jsonResponse))

		} else {
			// If there are no public rooms, create a new room and assign client to it
			if publicRoomBucket.IsEmpty() {
				newRoomID = utils.GetKey()
				newClientInfo.RoomID = newRoomID
				newRoom, err := AddAndUpdatePublicRooms([]model.ClientInfo{}, []model.ClientInfo{}, newClientInfo, newRoomID)
				if err != nil {
					log.Printf("Max no. of clients reached")
					return
				}
				log.Printf("User has been assigned %v\nNo rooms in PQ, creating new room..\n", newRoomID)
				jsonResponse, err := json.Marshal(model.ServerResponse{ResponseType: "total", ClientInfo: newClientInfo, RoomInfo: *newRoom, Point: model.Point{}})
				if err != nil {
					log.Printf("error parsing json")
					return
				}
				s.Write([]byte(jsonResponse))
				log.Printf("Sent client %v", string(jsonResponse))

			} else {
				newRoomID = publicRoomBucket.GetRoomID()
				prevRoomWithoutUpdate := publicRoomsMap[newRoomID]
				newClientInfo.RoomID = newRoomID
				log.Printf("Room alloted: %v\n", newRoomID)
				// update the groups of the current room which has the lowest no. of clients
				newRoom, err := AddAndUpdatePublicRooms(prevRoomWithoutUpdate.Group1, prevRoomWithoutUpdate.Group2, newClientInfo, newRoomID)
				if err != nil {
					log.Printf("Max no. of clients reached")
					return
				}

				if totalClientsInSession[newRoomID] == nil {
					totalClientsInSession[newRoomID] = make(map[string]*melody.Session)
				}
				totalClientsInSession[newRoomID][clientID] = s
				BroadcastMessageInRoom(newRoom, newClientInfo, model.Point{}, "total")
				log.Printf("User has been assigned %v\nNo need to create new Room, assigned to already exsiting\n", newRoomID)

			}

		}
		// update the info of the current session with its roomID and user name
		s.Set("info", &newClientInfo)

		// mapping the clientID with the sessions
		if totalClientsInSession[newRoomID] == nil {
			totalClientsInSession[newRoomID] = make(map[string]*melody.Session)
		}
		totalClientsInSession[newRoomID][clientID] = s
	} else if response.ResponseType == "connect" {
		// check if the roomID exists
		roomID := clientResponse.RoomID
		log.Printf("User wants to join room: %v\n", roomID)
		log.Printf("Private rooms: %v\n", privateRoomsMap)
		fmt.Printf("room: %v - Map: %v\n", roomID, privateRoomsMap[roomID])
		if _, ok := privateRoomsMap[roomID]; !ok {
			log.Printf("given room doesn't exists")
			return
		}
		//	check if the room already has 10 members
		totalClients := len(privateRoomsMap[roomID].Group1) + len(privateRoomsMap[roomID].Group2)
		if totalClients == 10 {
			log.Printf("max no. of clients reached")
			return
		}
		newClientInfo.RoomID = roomID
		s.Set("info", &newClientInfo)
		grp1, grp2 := utils.InsertClientInRoom(privateRoomsMap[roomID].Group1, privateRoomsMap[roomID].Group2, newClientInfo)
		newRoom := model.Room{RoomID: roomID, Group1: grp1, Group2: grp2}
		privateRoomsMap[roomID] = &newRoom
		if totalClientsInSession[roomID] == nil {
			totalClientsInSession[roomID] = make(map[string]*melody.Session)
		}
		totalClientsInSession[roomID][clientID] = s
		BroadcastMessageInRoom(&newRoom, newClientInfo, model.Point{}, "total")
	} else if clientResponse.ReponseType == "move" {
		currentClient := clientResponse.ClientInfo
		points := clientResponse.Point
		log.Printf("Current Client: %v Points: %v", currentClient, points)
		drawnPoints[currentClient.RoomID] = append(drawnPoints[currentClient.RoomID], points)
		if clientResponse.RoomType == "private" {
			BroadcastMessageInRoom(privateRoomsMap[currentClient.RoomID], currentClient, points, "move")
		} else {
			BroadcastMessageInRoom(publicRoomsMap[currentClient.RoomID], currentClient, points, "move")
		}

	}
}

// updates the groups with equal distribution, inserts the updated room in the priority queue
// and update the public room in the map
func AddAndUpdatePublicRooms(group1, group2 []model.ClientInfo, client model.ClientInfo, newRoomID string) (*model.Room, error) {
	log.Printf("Current client %v", client)
	client.RoomID = newRoomID
	grp1, grp2 := utils.InsertClientInRoom(group1, group2, client)
	if len(grp1)+len(grp2) > 10 {
		log.Printf("No. of clients more than 10")
		return &model.Room{}, errors.New("more than permitted clients")
	}
	newRoom := model.Room{RoomID: newRoomID, Group1: grp1, Group2: grp2}
	log.Printf("New room: %v", newRoom)
	publicRoomBucket.AddUserToBucket(client.RoomID)
	// update the mapping for public room
	publicRoomsMap[newRoomID] = &newRoom
	log.Printf("Allocated new room: %v\n", newRoom)
	return &newRoom, nil
}

// broadcast message in a room
func BroadcastMessageInRoom(room *model.Room, clientInfo model.ClientInfo, pointInfo model.Point, responseType string) {
	// broadcast in group1
	if room == nil {
		log.Printf("No room to broadcast")
		return
	}
	log.Printf("Current room while broadcasting %v", room)
	if room.Group1 != nil {
		for _, client := range room.Group1 {
			if clientInfo.ClientID == client.ClientID && responseType == "move" {
				continue
			}
			session := totalClientsInSession[room.RoomID][client.ClientID]
			serverResponse := model.ServerResponse{ResponseType: responseType, ClientInfo: clientInfo, RoomInfo: *room, Point: pointInfo}
			jsonReponse, err := json.Marshal(&serverResponse)
			if err != nil {
				log.Printf("cant parse json response")
				return
			}
			session.Write([]byte(jsonReponse))
		}
	}

	log.Printf("Broadcasted msg in grp %v\n", room.Group1)
	if room.Group2 != nil {
		for _, client := range room.Group2 {
			if clientInfo.ClientID == client.ClientID && responseType == "move" {
				continue
			}
			session := totalClientsInSession[room.RoomID][client.ClientID]
			serverResponse := model.ServerResponse{ResponseType: responseType, ClientInfo: clientInfo, RoomInfo: *room, Point: pointInfo}
			jsonReponse, err := json.Marshal(&serverResponse)
			if err != nil {
				log.Printf("cant parse json response")
				return
			}
			session.Write([]byte(jsonReponse))
		}
	}

	log.Printf("Broadcasted msg in grp %v\n", room.Group2)

}

// When user diconnects, these logic updates everything accordingly
func UpdateEverythingAfterDisconnect(client model.ClientInfo, roomID string, roomType string) {
	var room map[string]*model.Room
	if roomType == "public" {
		room = publicRoomsMap
	} else {
		room = privateRoomsMap
	}
	currentRoomStatus := room[roomID]
	if currentRoomStatus == nil {
		log.Printf("Room is empty\n")
		return
	}
	// len := len(currentRoomStatus.Group1) + len(currentRoomStatus.Group2)
	log.Printf("Current Room: %v", currentRoomStatus)
	var newGrp1 []model.ClientInfo = make([]model.ClientInfo, 0)
	var newGrp2 []model.ClientInfo = make([]model.ClientInfo, 0)
	var err error
	//	TODO: Handle when no. of clients in a room becomes < 4
	// if len < 4 {
	// 	log.Fatal("No. of clients < 4, can't play the game")
	// }
	if currentRoomStatus.Group1 != nil {
		newGrp1 = currentRoomStatus.Group1
	}
	if currentRoomStatus.Group2 != nil {
		newGrp2 = currentRoomStatus.Group2
	}
	newGrp1, err = utils.Remove(newGrp1, client)
	if err != nil {
		newGrp2, err = utils.Remove(newGrp2, client)
		if err != nil {
			log.Printf("client is in neither of rooms")
		}
	}
	newRoom := model.Room{RoomID: client.RoomID, Group1: newGrp1, Group2: newGrp2}
	// delete the user from privateRoomMap
	delete(room, roomID)

	// delete from totalConnectedClientsMap
	delete(totalClientsInSession[roomID], client.ClientID)

	// update the room bucket with size
	if roomType == "public" {
		publicRoomBucket.RemoveUserFromBucket(roomID)
	}

	// broadcast others that user has disconnected
	BroadcastMessageInRoom(&newRoom, client, model.Point{}, "dis")
}
