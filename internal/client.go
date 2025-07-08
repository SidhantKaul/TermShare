package internal

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var userid string
var is_this_client_editor bool

var wg sync.WaitGroup

func SetUpClient(pPort string, username string, pIsEditor bool) (bool, error) {

	is_this_client_editor = pIsEditor

	conn, err := net.Dial("tcp", pPort)

	if err != nil {
		log.Printf("Failed to listen on port %s: %v\n", pPort, err)

		return false, err
	}

	defer conn.Close() // close the connection before returning

	fmt.Fprintf(conn, "%s %s\n", MsgHello, username)

	//if server does not reply within 5 seconds, it means server is busy try to connect again
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	ok, err := RecieveAcknowledgement(conn)
	if err != nil {
		log.Printf("Some problem with the Acknolwdgment from the server")

	}

	if !ok { //username is already taken
		return false, nil
	}

	wg.Add(1)

	//Listent to the broadcast

	go DisplayBroadCast(conn)

	//Now if the client wants to write

	go Write(conn)

	wg.Wait()

	return true, nil
}

//Listen to the broadcast

func DisplayBroadCast(pConnection net.Conn) {

	defer wg.Done()

	reader := bufio.NewReader(pConnection)

	for {
		line, err := reader.ReadString('\n')

		if err != nil { // If the connection to Host has abruptly closed need to stop listening and exit.
			return
		}

		clean := strings.TrimSpace(line)
		parts := strings.SplitN(clean, "=", 2)

		if len(parts) == 2 && parts[0] == KeyType {
			switch parts[1] {
			case MsgTypeDenyControl:
				fmt.Print("Your editor request has been rejected\n")

			case MsgTypeGrantControl:
				is_this_client_editor = true
				fmt.Print("Your editor request has been accepted\n")

			case MsgTypeControlGivenBack:
				is_this_client_editor = false
				fmt.Print("Editor control has reliquished\n")
			case MsgTypeQuitApproved:
				return

			default:
				fmt.Printf("%s\n", line) // show on client's terminal
			}
		} else {
			fmt.Printf("%s\n", line) // show on client's terminal
		}
	}

}

// listen if server sends an acknowledgement for the HI
func RecieveAcknowledgement(pConnection net.Conn) (bool, error) {

	reader := bufio.NewReader(pConnection)

	line, err := reader.ReadString('\n')

	if err != nil {
		log.Printf("Not properly formated")
		return false, err
	}

	clean := strings.TrimSpace(line)

	parts := strings.SplitN(clean, " ", 2)

	if len(parts) == 2 && parts[0] == "WELCOME" {

		userid = strings.TrimSpace(parts[1])

		pConnection.SetDeadline(time.Time{}) // ack rcvd from server

		fmt.Printf("You have joined the Host\n")

	} else {
		// Handle malformed message

		parts = strings.Split(clean, "=") //Check if a protocol msg was sent

		if parts[1] == MsgTypeClientNameTaken {

			fmt.Printf("This Client Name is taken, Please use another one.\n")
			return false, nil
		}

		log.Printf("There seems to be some problem with the Acknowledgment packet\n")
		return false, nil
	}

	return true, nil
}

func Write(pConnection net.Conn) {

	inputScanner := bufio.NewScanner(os.Stdin)

	for inputScanner.Scan() {
		text := inputScanner.Text()

		// If client types a control command:
		switch {
		case text == "/request_control":
			SendProtocolMsgToServer(MsgTypeRequestControl, pConnection)
		case text == "/give_back_control":
			SendProtocolMsgToServer(MsgTypeGiveUpControl, pConnection)

		case text == "/quit":
			SendProtocolMsgToServer(MsgTypeQuit, pConnection)

		case is_this_client_editor:
			fmt.Fprint(pConnection, text+"\n")
		}
	}
}

func SendProtocolMsgToServer(msg string, pConnection net.Conn) {
	input_text := "type=" + msg + "\n"

	fmt.Fprint(pConnection, input_text)
}
