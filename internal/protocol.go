package internal

const (
	MsgTypeRequestControl   = "request_control"
	MsgTypeGrantControl     = "grant_control"
	MsgTypeDenyControl      = "deny_control"
	MsgTypeGiveUpControl    = "give_back_control"
	MsgTypeControlGivenBack = "control_given_back"
	MsgTypeControlUpdate    = "control_update" // to notify all clients
	MsgTypeQuit             = "quit"
	MsgTypeClientNameTaken  = "client_name_taken"
	MsgTypeQuitApproved     = "Leave"
	MsgTypeAccept           = "y"
	MsgTypeReject           = "n"
)

const (
	MsgHello          = "HELLO"
	MsgWelcome        = "WELCOME"
	KeyType           = "type"
	DefaultClientName = "HOST"
)
