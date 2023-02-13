package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"google.golang.org/grpc"
	"grpc/message"
	"io/ioutil"
	"log"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:12345", grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	client := message.NewAlgorithmsClient(conn)

	fName := "./frog.jpg"
	input, _ := ioutil.ReadFile(fName)
	imgStr := base64.StdEncoding.EncodeToString(input)

	resp, err := client.Label(context.Background(), &message.LabelRequest{Base64Img: imgStr})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}

	fmt.Printf("########### %+v\n", resp)

	defer conn.Close()
}
