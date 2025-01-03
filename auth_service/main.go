package main

import (
	// "flag"
	"auth-service/infra"
	"log"
	"net"

	pb "auth-service/proto"

	"google.golang.org/grpc"
)

func main() {
	ctx, err := infra.NewServiceContext()
	if err != nil {
		log.Fatal(err)
	}

	// if shouldNotLaunchServer() {
	// 	return
	// }

	var listener net.Listener
	listener, err = net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &ctx.Service.Auth)
	if err := s.Serve(listener); err != nil {
		log.Fatal(err)
	}
	log.Println("Server running on port 50051")
}

// func shouldNotLaunchServer() bool {
// 	shouldNotLaunch := false

// 	flag.Parse()
// 	flag.Visit(func(f *flag.Flag) {
// 		if f.Name == "m" {
// 			shouldNotLaunch = true
// 		}

// 		if f.Name == "s" {
// 			shouldNotLaunch = true
// 		}
// 	})

// 	return shouldNotLaunch
// }
