package grpc

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"user-service/config"
	"user-service/models"
	"user-service/proto"
	"user-service/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	userService service.UserService
	proto.UnimplementedUserServiceServer
}

func NewServer(userService service.UserService) *server {
	return &server{userService: userService}
}

func (s *server) GetAllUsers(ctx context.Context, req *proto.GetAllUsersRequest) (*proto.UsersList, error) {
	name := req.GetName()
	users, err := s.userService.GetAllUsers(name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get all users: %v", err)
	}
	var userResponses []*proto.User
	for _, user := range users {
		userResponses = append(userResponses, &proto.User{
			Id:        int32(user.ID),
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			IsOnline:  user.IsOnline,
		})

	}
	return &proto.UsersList{Users: userResponses}, nil
}

func (s *server) UpdateUser(ctx context.Context, req *proto.UpdateUserRequest) (*proto.UpdateUserResponse, error) {

	if req.FirstName == "" || req.LastName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "First name or last name cannot be empty")
	}
	if len(req.FirstName) < 2 || len(req.LastName) < 2 {
		return nil, status.Errorf(codes.InvalidArgument, "First name or last name min 2 characters")
	}

	existingUser, err := s.userService.GetUserInfo(strconv.Itoa(int(req.Id)))
	if err != nil {
		if err.Error() == "user not found" {
			return nil, status.Errorf(codes.NotFound, "User with ID %d not found", req.Id)
		}
		log.Printf("Error fetching user info: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to fetch user info")
	}

	user := &models.User{
		ID:        existingUser.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	if err := s.userService.UpdateUser(user); err != nil {
		log.Printf("Error updating user: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to update user: %v", err)
	}

	log.Printf("User with ID %d updated successfully", req.Id)
	return &proto.UpdateUserResponse{Message: "Update success"}, nil
}

func StartGRPCServer(userService service.UserService) {
	config.LoadEnv()
	grpcPort := os.Getenv("GRPC_PORT")
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	proto.RegisterUserServiceServer(grpcServer, NewServer(userService))
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
