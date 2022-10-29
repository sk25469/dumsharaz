package utils

type RoomBucket struct {
	//	Bucket of size 10, to have atmost 9 clients in a room, as soon the clients become 10, it won't matter
	Buckets [10]map[string]bool
}

func Init() *RoomBucket {
	var roomBucket RoomBucket
	for i := 0; i < 10; i++ {
		roomBucket.Buckets[i] = make(map[string]bool)
	}
	return &roomBucket
}

// A client has connected to roomID, we need to remove the room from its existing bucket and put in updated sized bucket
func (bucket RoomBucket) AddUserToBucket(roomID string) {
	bucketSize := -1
	for i := 0; i < 10; i++ {
		if _, ok := bucket.Buckets[i][roomID]; ok {
			bucketSize = i
			break
		}
	}
	//	remove room from current sized bucket
	delete(bucket.Buckets[bucketSize], roomID)

	//	add room to the bucket of 1 greater size
	bucket.Buckets[bucketSize+1][roomID] = true
}

// A client has disconnected from roomID, we need to remove the room from its existing bucket and put in updated sized bucket
func (bucket RoomBucket) RemoveUserFromBucket(roomID string) {
	bucketSize := -1
	for i := 1; i < 10; i++ {
		if _, ok := bucket.Buckets[i][roomID]; ok {
			bucketSize = i
			break
		}
	}
	//	remove room from current sized bucket
	delete(bucket.Buckets[bucketSize], roomID)

	//	add room to the bucket of 1 smaller size
	bucket.Buckets[bucketSize-1][roomID] = true
}

// Check if all the buckets are empty
func (bucket RoomBucket) IsEmpty() bool {
	var empty bool = true
	for i := 0; i < 10; i++ {
		if len(bucket.Buckets[i]) > 0 {
			empty = false
			break
		}
	}
	return empty
}

// Get roomId of the greatest sized bucket (preferablly sized 9)
func (bucket RoomBucket) GetRoomID() string {
	for i := 9; i >= 0; i-- {
		if len(bucket.Buckets[i]) > 0 {
			return GetRoomIDFromMap(bucket.Buckets[i])
		}
	}
	return ""
}

func GetRoomIDFromMap(roomMap map[string]bool) string {
	for key := range roomMap {
		return key
	}
	return ""
}
