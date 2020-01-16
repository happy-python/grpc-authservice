package main

import (
	"context"
	"log"
	"net"
	"os"

	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2"
	v2alpha "github.com/datawire/ambassador/pkg/api/envoy/service/auth/v2alpha"
	_type "github.com/datawire/ambassador/pkg/api/envoy/type"
	"google.golang.org/grpc"
	rpc "istio.io/gogo-genproto/googleapis/google/rpc"
)

var Address = getEnv("ADDRESS", ":20020")

// 实现gRPC服务来处理认证
type Server struct {
	Authorized   *pb.CheckResponse // 已认证 code: rpc 0 -> http 200
	Unauthorized *pb.CheckResponse // 未认证 code: rpc 16 -> http 401
	Forbidden    *pb.CheckResponse // 拒绝 code: rpc 7 -> http 403
	Unavailable  *pb.CheckResponse // 服务不可用 code: rpc 14 -> http 503
}

// 检查请求
func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	log.Println("ACCESS",
		req.GetAttributes().GetRequest().GetHttp().GetMethod(),
		req.GetAttributes().GetRequest().GetHttp().GetHost(),
		req.GetAttributes().GetRequest().GetHttp().GetPath(),
		req.GetAttributes().GetRequest().GetHttp().GetQuery(),
		req.GetAttributes().GetRequest().GetHttp().GetFragment(),
	)

	// 获取请求头
	headers := req.GetAttributes().GetRequest().GetHttp().GetHeaders()

	log.Printf("authorization: %s\n", headers["authorization"])

	if headers["authorization"] == "" {
		log.Println("missing header authorization")
		return s.Unauthorized, nil
	} else if headers["authorization"] != "123" {
		log.Println("wrong header authorization")
		return s.Forbidden, nil
	}

	return s.Authorized, nil
}

// 服务初始化并运行
func Run() error {
	listen, err := net.Listen("tcp", Address)
	if err != nil {
		return err
	}

	instance := &Server{}

	// Authorized returns an response object with status OK and a header
	// that should be sent to the upstream service.
	instance.Authorized = &pb.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.OK)},
		HttpResponse: &pb.CheckResponse_OkResponse{
			OkResponse: &pb.OkHttpResponse{
				Headers: []*core.HeaderValueOption{
					{
						Header: &core.HeaderValue{
							Key:   "x-ok",
							Value: "this will be sent to the upstream service",
						},
					},
				},
			},
		},
	}

	// Unauthorized will return header, status code and a body to the
	// downstream client.
	instance.Unauthorized = &pb.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.UNAUTHENTICATED)},
		HttpResponse: &pb.CheckResponse_DeniedResponse{
			DeniedResponse: &pb.DeniedHttpResponse{
				Status: &_type.HttpStatus{
					Code: _type.StatusCode_Unauthorized,
				},
				Headers: []*core.HeaderValueOption{
					{
						Header: &core.HeaderValue{
							Key:   "x-failed",
							Value: "this will be sent to the client",
						},
					},
				},
				Body: "Unauthorized",
			},
		},
	}

	// Forbidden will return header, status code and a body to the
	// downstream client.
	instance.Forbidden = &pb.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.PERMISSION_DENIED)},
		HttpResponse: &pb.CheckResponse_DeniedResponse{
			DeniedResponse: &pb.DeniedHttpResponse{
				Status: &_type.HttpStatus{
					Code: _type.StatusCode_Forbidden,
				},
				Headers: []*core.HeaderValueOption{
					{
						Header: &core.HeaderValue{
							Key:   "x-failed",
							Value: "this will be sent to the client",
						},
					},
				},
				Body: "Forbidden",
			},
		},
	}

	// 503 Service Unavailable
	instance.Unavailable = &pb.CheckResponse{
		Status: &rpc.Status{Code: int32(rpc.UNAVAILABLE)},
		HttpResponse: &pb.CheckResponse_DeniedResponse{
			DeniedResponse: &pb.DeniedHttpResponse{
				Status: &_type.HttpStatus{
					Code: _type.StatusCode_ServiceUnavailable,
				},
				Headers: []*core.HeaderValueOption{
					{
						Header: &core.HeaderValue{
							Key:   "x-failed",
							Value: "this will be sent to the client",
						},
					},
				},
				Body: "Unavailable",
			},
		},
	}

	server := grpc.NewServer()
	v2alpha.RegisterAuthorizationServer(server, instance)
	log.Printf("serving on port %s", Address)
	if err = server.Serve(listen); err != nil {
		return err
	}

	return nil
}

// 获取环境变量
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	if err := Run(); err != nil {
		log.Printf("run err: %v", err)
	}
}
