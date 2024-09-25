package speech

import (
	"context"
	"fmt"
	"time"

	auth "github.com/vito-ai/auth"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	pb "github.com/vito-ai/go-genproto/vito-openapi/stt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type gRPCClient struct {
	endpoint string
	coonPool *grpc.ClientConn
	client   pb.OnlineDecoderClient

	tp auth.TokenProvider
}

// gRPC 스트리밍을 위한 새로운 gRPC 클라이언트를 만듭니다.
func NewStreamingClient(opts ...auth.Option) (*gRPCClient, error) {
	tp, err := auth.NewRTZRTokenProvider(opts...)
	if err != nil {
		return nil, err
	}

	c := &gRPCClient{
		endpoint: "grpc-openapi.vito.ai:443",
		tp:       tp,
	}

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))) //TLS certificate nil로 해서 header를 통한 authority 진행

	conn, err := grpc.NewClient(c.endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	newCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if isConnReady := conn.WaitForStateChange(newCtx, connectivity.Ready); !isConnReady {
		return nil, fmt.Errorf("cannot connect to vito grpc server")
	}

	client := pb.NewOnlineDecoderClient(conn)

	c.coonPool = conn
	c.client = client
	return c, nil
}

func (c *gRPCClient) StreamingRecognize(ctx context.Context) (pb.OnlineDecoder_DecodeClient, error) {
	token, err := c.tp.Token(ctx)
	if err != nil {
		return nil, err
	}
	md := metadata.Pairs("authorization", fmt.Sprintf("%s %v", "bearer", token.AccessToken))
	ctxWithAuth := metautils.NiceMD(md).ToOutgoing(ctx)

	stream, err := c.client.Decode(ctxWithAuth)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (c *gRPCClient) Close() error {
	return c.coonPool.Close()
}
