package telegram

// awaitingAnswer maps admin chat ID to the question ID they are answering.
var awaitingAnswer = make(map[int64]int64)

// SetAwaitingAnswer sets that this admin chat is writing an answer for the given question ID.
func SetAwaitingAnswer(adminChatID, questionID int64) {
	awaitingAnswer[adminChatID] = questionID
}

// GetAwaitingAnswer returns the question ID the admin is answering, if any.
func GetAwaitingAnswer(adminChatID int64) (questionID int64, ok bool) {
	q, ok := awaitingAnswer[adminChatID]
	return q, ok
}

// ClearAwaitingAnswer clears the answer state for this admin chat.
func ClearAwaitingAnswer(adminChatID int64) {
	delete(awaitingAnswer, adminChatID)
}
