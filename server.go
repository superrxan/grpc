package main

import (
	"context"
	"encoding/base64"
	"google.golang.org/grpc"
	"grpc/labelImage"
	"grpc/message"
	"log"
	"net"
)

func init() {
	labelImage.LabelImage.Init()
}

type LabelService struct {
	*message.UnimplementedAlgorithmsServer
}

func (l *LabelService) Label(_ context.Context, in *message.LabelRequest) (*message.LabelReply, error) {
	imgByte, _ := base64.StdEncoding.DecodeString(in.Base64Img)

	return labelImage.LabelImage.Exec(imgByte)
}

func main() {
	srv := grpc.NewServer()
	message.RegisterAlgorithmsServer(srv, &LabelService{})

	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	err = srv.Serve(listener)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
