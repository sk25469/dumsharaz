package model

type ClientInfo struct {
	RoomID   string `json:"room_id"`
	ClientID string `json:"client_id"`

	//	!FIXME: No need of x and y in clientInfo right now
	X    string `json:"x"`
	Y    string `json:"y"`
	Name string `json:"name"`
}
