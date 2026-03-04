package telegram

// awaitingBroadcastChatIDs tracks admin chat IDs that pressed "Send to all" and are expected to send the message next.
var awaitingBroadcastChatIDs = make(map[int64]struct{})

// SetAwaitingBroadcast marks the chat as waiting for the broadcast message text.
func SetAwaitingBroadcast(chatID int64) {
	awaitingBroadcastChatIDs[chatID] = struct{}{}
}

// IsAwaitingBroadcast reports whether the chat is waiting for a broadcast message.
func IsAwaitingBroadcast(chatID int64) bool {
	_, ok := awaitingBroadcastChatIDs[chatID]
	return ok
}

// ClearAwaitingBroadcast removes the chat from awaiting-broadcast state.
func ClearAwaitingBroadcast(chatID int64) {
	delete(awaitingBroadcastChatIDs, chatID)
}
