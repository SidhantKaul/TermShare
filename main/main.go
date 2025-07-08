package main

import (
	"TermShare/internal"
	"fmt"
)

func main() {
	var choice int
	var addr string
	var client_name string

	fmt.Printf("How do you want to initialize as a Host or Client\n")
	fmt.Printf("1. Host\n")
	fmt.Printf("2. Client\n")

	fmt.Scanf("%d", &choice)

	if choice == 1 {

		fmt.Printf("Enter the I.P on which you want to listen for request\n")
		fmt.Scanf("%s\n", &addr)

		server := internal.NewServer(addr)
		server.SetUpHost()

	} else {

		fmt.Printf("Enter the Host address\n")
		fmt.Scanf("%s", &addr)

		for {
			fmt.Printf("Enter name of the Client\n")
			fmt.Scanf("%s\n", &client_name)

			ok, err := internal.SetUpClient(addr, client_name, false)

			if err != nil {
				fmt.Printf("Issue faced while connecting to client: %s", err.Error())

			} else if ok {
				break
			}
		}

	}

	// if choice == 1 {

	// 	var name string
	// 	fmt.Printf("Give your client a name\n")

	// 	fmt.Scanf("%s", &name)

	// 	internal.SetUpClient("127.0.0.1:9090", name, false)

	// } else {
	// 	server := internal.NewServer("127.0.0.1:9090")
	// 	server.SetUpHost()
	// }

	//127.0.0.1:9090
}
