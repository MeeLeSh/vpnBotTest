package telegram

// awaitingHelpChatIDs tracks chat IDs that have pressed Help and are expected to send a question next.
var awaitingHelpChatIDs = make(map[int64]struct{})

// SetAwaitingHelpQuestion marks the chat as waiting for a help question.
func SetAwaitingHelpQuestion(chatID int64) {
	awaitingHelpChatIDs[chatID] = struct{}{}
}

// IsAwaitingHelpQuestion reports whether the chat is waiting for a help question.
func IsAwaitingHelpQuestion(chatID int64) bool {
	_, ok := awaitingHelpChatIDs[chatID]
	return ok
}

// ClearAwaitingHelpQuestion removes the chat from awaiting-help state.
func ClearAwaitingHelpQuestion(chatID int64) {
	delete(awaitingHelpChatIDs, chatID)
}
