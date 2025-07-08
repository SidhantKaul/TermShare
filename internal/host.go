package internal

import (
	"TermShare/pty"
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

//Client

type Client struct {
	Username  string
	SessionID string
	Conn      net.Conn
	Send      chan string // optional: to send data to client
}

type ClientManager struct {

	// Client Tracker
	connectedUsers sync.Map

	// Host's Client
	defaultClient string

	// Current Editor
	currentEditor string

	mu sync.Mutex
}

type Server struct {
	editor_controller chan bool

	exit_listening_loop chan bool

	//ptywriter *PTYBroadCast
	ptywriter *PTYBroadCast

	clientMgr *ClientManager

	//out bash pty session
	ptysession *pty.PTYSession

	port string
}

//PTY Broadcaster

type PTYBroadCast struct {
	server *Server
}

//Acknowledgmenet

func NewServer(pPort string) *Server {

	var err error

	clientMgr := &ClientManager{
		defaultClient: DefaultClientName,
		currentEditor: DefaultClientName,
	}

	server := Server{
		editor_controller:   make(chan bool),
		exit_listening_loop: make(chan bool),
		ptywriter:           &PTYBroadCast{},
		clientMgr:           clientMgr,
		port:                pPort,
	}

	// Launch a default client
	go server.SetUpDefaultClient(server.port)

	server.ptywriter.server = &server

	server.ptysession, err = pty.CreatePtySession(server.ptywriter)

	if err != nil {
		log.Printf("Not able to create a pty terminal session: %s\n", err.Error())
		return nil
	}

	return &server
}

func (server *Server) ShutDown() {

	server.CloseAllTheConnections()

	close(server.editor_controller)
	close(server.exit_listening_loop)

	server.ptysession.ClosePtySession()
}

func (server *Server) SetUpHost() {
	ln, err := net.Listen("tcp", server.port)
	if err != nil {
		log.Printf("Failed to listen on port %s: %v\n", server.port, err)
		return
	}

	defer func() {
		ln.Close()
		server.ShutDown()
	}()

	connChan := make(chan net.Conn)
	errChan := make(chan error)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errChan <- err
				return
			}
			connChan <- conn
		}
	}()

	for {
		select {
		case <-server.exit_listening_loop:
			log.Printf("Exit signal received. Stopping accept loop.")
			return
		case err := <-errChan:
			log.Printf("Accept error: %v", err)
		case conn := <-connChan:
			go server.HandleConnection(conn)
		}
	}
}

func (server *Server) HandleConnection(conn net.Conn) {

	//We begin with verifying the handshake

	err, username := server.VerifyHandShake(conn)

	if !err {
		return
	}

	//Once Handshake verfied, we send an ack

	if !server.SendAcknowledgement(conn, username) {
		return
	}

	val, ok := server.clientMgr.connectedUsers.Load(username)

	if !ok {
		log.Printf("User %s not found", username)
		return
	}

	client := val.(Client)

	//Start Listening to broadcast

	go server.StartListeningToBroadCast(&client)

	//Now we we start the broadcast if any

	go server.StartListeningToClientInput(conn, &client)

}

func (server *Server) VerifyHandShake(conn net.Conn) (bool, string) {
	var username string

	//We timeout if HELLO not recived in 5 secs
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')

	if err != nil {
		log.Printf("Not properly formated")
		return false, ""
	}

	clean := strings.TrimSpace(line)

	parts := strings.SplitN(clean, " ", 2)

	if len(parts) == 2 && parts[0] == MsgHello {
		username = strings.TrimSpace(parts[1])
	} else {
		// Handle malformed message

		log.Printf("There seems to be some problem with the packet")
		return false, ""
	}

	conn.SetDeadline(time.Time{}) //Handshake verifed

	return true, username
}

func (server *Server) SendAcknowledgement(conn net.Conn, username string) bool {
	session_id := uuid.New().String()

	ack_to_be_sent := fmt.Sprintf("%s %s\n", MsgWelcome, session_id)

	_, usernameExists := server.clientMgr.connectedUsers.Load(username)

	if usernameExists {

		SendDirectedProtocolMessage(conn, MsgTypeClientNameTaken)

		return false
	}

	_, err := fmt.Fprint(conn, ack_to_be_sent)

	if err != nil {
		log.Printf("Failed to send WELCOME to %s: %v\n", username, err)
		return false
	}

	client := Client{username, session_id, conn, make(chan string, 100)}

	server.clientMgr.connectedUsers.Store(username, client)

	log.Printf("New User: %s Added\n", username)

	return true
}

func (server *Server) StartBroadCasting(msg string, current_editor_name string) {

	displaye_msg := "[" + current_editor_name + "]" + " " + msg + "\n"

	server.clientMgr.connectedUsers.Range(func(key, value interface{}) bool {
		id := key.(string)
		client := value.(Client)

		if id != current_editor_name {
			select {
			case client.Send <- displaye_msg: // if the client's rcvr is full just drop the msg
			default:
			}
		}
		return true
	})
}

func (server *Server) StartListeningToBroadCast(client *Client) {

	defer server.RemoveClient(client)

	for {
		msg, ok := <-client.Send

		if !ok {
			log.Printf("Send channel closed for %s", client.Username)
			return
		}
		_, err := fmt.Fprint(client.Conn, msg)

		if err != nil {
			log.Printf("Failed to send to %s: %v", client.Username, err)
			return
		}
	}

}

/* We get input from a client in only 2 cases
*  If the client is sending a PROTOCOL MSG
*  else the client is now an editor
 */

func (server *Server) StartListeningToClientInput(conn net.Conn, client *Client) {
	defer server.RemoveClient(client)

	reader := bufio.NewReader(conn)

	for {
		isProtocolMsg := false

		line, err := reader.ReadString('\n')
		if err != nil {

			log.Printf("Error reading from %s: %v", client.Username, err)
			break
		}

		clean := strings.TrimSpace(line)
		parts := strings.SplitN(clean, "=", 2)

		val := ""
		if len(parts) == 2 && parts[0] == "type" {
			isProtocolMsg = true
			val = parts[1]
		}

		defaultClientVal, _ := server.clientMgr.connectedUsers.Load(server.clientMgr.defaultClient)
		defaultClient := defaultClientVal.(Client)

		switch {

		case isProtocolMsg:

			switch {
			case val == MsgTypeRequestControl:
				needToValidateRequest := true

				server.clientMgr.mu.Lock()
				if server.clientMgr.currentEditor != server.clientMgr.defaultClient {
					needToValidateRequest = false
					SendDirectedProtocolMessage(client.Conn, MsgTypeDenyControl)
				}
				if needToValidateRequest {
					GiveEditorAccess(client)
					go server.HandOverEditorControl(client, defaultClient)
				}
				server.clientMgr.mu.Unlock()

			case val == MsgTypeGiveUpControl:
				server.clientMgr.mu.Lock()
				server.clientMgr.currentEditor = server.clientMgr.defaultClient
				server.clientMgr.mu.Unlock()

				SendDirectedProtocolMessage(conn, MsgTypeControlGivenBack)
				SendDirectedProtocolMessage(defaultClient.Conn, MsgTypeGrantControl)
				server.BroadcastControlChange(server.clientMgr.defaultClient)

			case val == MsgTypeQuit:

				server.clientMgr.mu.Lock()

				if client.Username == server.clientMgr.defaultClient {

					server.exit_listening_loop <- true

				} else {
					SendDirectedProtocolMessage(client.Conn, MsgTypeQuitApproved)

					server.RemoveClient(client)

					server.BroadcastControlChange(server.clientMgr.defaultClient)

				}

				server.clientMgr.mu.Unlock()
				return
			}

		case server.clientMgr.defaultClient == client.Username && clean == MsgTypeAccept:
			server.editor_controller <- true

		case server.clientMgr.defaultClient == client.Username && clean == MsgTypeReject:
			server.editor_controller <- false

		default:

			server.StartBroadCasting(clean, server.clientMgr.currentEditor)
			server.ptysession.FeedInput(clean, server.ptywriter)

		}
	}
}

func (server *Server) HandOverEditorControl(client *Client, defaul_client Client) {

	accept := <-server.editor_controller

	if !accept {

		SendDirectedProtocolMessage(client.Conn, MsgTypeDenyControl)
		return
	}

	//Here we let the host choose if he wants to give editor access to requistor

	if server.clientMgr.currentEditor == server.clientMgr.defaultClient {

		if server.clientMgr.currentEditor == client.Username { //default client can't have control
			return
		}

		SendDirectedProtocolMessage(defaul_client.Conn, MsgTypeControlGivenBack) //Letting the default client know that editor will be changed

		server.clientMgr.currentEditor = client.Username

		SendDirectedProtocolMessage(client.Conn, MsgTypeGrantControl) // Let the requestee know his request has been accepted
		server.BroadcastControlChange(client.Username)                //BroadCast to everyone that editor has changed

	} else {
		SendDirectedProtocolMessage(client.Conn, MsgTypeDenyControl)

	}

}

// block until the host provides an input
func GiveEditorAccess(client *Client) {

	log.Printf("\nUser: %s is requesting for editor access\n", client.Username)
	log.Printf("Enter 'y' if you want to grant the editor access, else 'n':")
}

func (writer *PTYBroadCast) Write(buffer []byte) (int, error) {

	buf_to_string := string(buffer)

	buf_to_string += "\n"

	writer.server.clientMgr.connectedUsers.Range(func(key, value interface{}) bool {
		client := value.(Client)

		select {
		case client.Send <- buf_to_string: // if the client's rcvr is full just drop the msg
		default:
		}

		return true
	})

	return len(buf_to_string), nil
}

func (server *Server) CloseAllTheConnections() {
	server.clientMgr.connectedUsers.Range(func(key, value interface{}) bool {
		client := value.(Client)

		SendDirectedProtocolMessage(client.Conn, MsgTypeQuitApproved)

		close(client.Send) // Close the send channel
		client.Conn.Close()

		return true
	})
}

func (server *Server) BroadcastControlChange(username string) {

	var empty_str string

	server.StartBroadCasting("Editor access has been granted to: "+username, username)

	server.ptysession.FeedInput(empty_str, &PTYBroadCast{}) // get terminal ready for new editor's input
}

func SendDirectedProtocolMessage(conn net.Conn, msg string) {

	input_text := "type=" + msg + "\n"

	fmt.Fprint(conn, input_text)
}

// wait for a dew seconds so that server is ready

func (server *Server) SetUpDefaultClient(pPort string) {

	time.Sleep(3 * time.Second)

	SetUpClient(pPort, server.clientMgr.defaultClient, true)
}

// close the connection and remove the client from the user's map
func (server *Server) RemoveClient(client *Client) {

	_, ok := server.clientMgr.connectedUsers.Load(client.Username)

	if !ok {
		return
	}

	client.Conn.Close()
	server.clientMgr.connectedUsers.Delete(client.Username)

	log.Printf("Stopped listening to %s", client.Username)
}
