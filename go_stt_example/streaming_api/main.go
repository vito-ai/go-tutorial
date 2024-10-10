package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	pb "github.com/vito-ai/go-genproto/vito-openapi/stt"
	"github.com/xfrr/goffmpeg/transcoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const ServerHost = "grpc-openapi.vito.ai:443" // rtzr grpc server
const ClientId = "YOUR_CLIENT_ID"
const ClientSecret = "YOUR_CLIENT_SECRET"

var False = false
var True = true

const SAMPLE_RATE int = 8000
const BYTES_PER_SAMPLE int = 2

/*
예제를 위해 파일을 읽기 위한 Interface 만들기
*/
type FileStreamer struct {
	file *os.File
}

// 정해진 byteSize (maxSize = 1024) 까지 파일을 읽어와서 전달한다.
func (fs *FileStreamer) Read(p []byte) (int, error) {
	byteSize := len(p)
	maxSize := 1024

	if byteSize > maxSize {
		byteSize = maxSize
	}

	// buffer가 터지는 것을 막기 위하여 delay 시킴.
	defer time.Sleep(time.Duration(byteSize/((SAMPLE_RATE*BYTES_PER_SAMPLE)/1000)) * time.Millisecond)
	return fs.file.Read(p[:byteSize])
}

// 파일 읽기가 끝나면 안정적으로 파일을 닫고 끝낸다.
func (fs *FileStreamer) Close() error {
	defer os.Remove(fs.file.Name())
	return fs.file.Close()
}

// 단순히 오디오 파일을 열 때만 필요함.
// 1. audio 파일을 연다.
// 2. 요구하는 속성에 맞게 파일을 변환한다.
// 3. 새롭게 만들어진 음성 파일을 반환한다.
func OpenAudioFile(audioFile string) (io.ReadCloser, error) {
	fileName := filepath.Base(audioFile)                                                                     // local에서 오디오 파일을 찾는다.
	i := strings.LastIndex(fileName, ".")                                                                    // 오디오 파일의 확장자를 찾는다.
	audioFileName8K := filepath.Join(os.TempDir(), fileName[:i]) + fmt.Sprintf("_%d.%s", SAMPLE_RATE, "wav") //결과 확장자를 wav 파일로 설정

	// //FFmpeg을 통해 음성 파일 변환
	trans := new(transcoder.Transcoder)
	if err := trans.Initialize(audioFile, audioFileName8K); err != nil {
		log.Fatal(err)
	}

	// //변환할 비디오 속성
	trans.MediaFile().SetAudioRate(SAMPLE_RATE)
	trans.MediaFile().SetAudioChannels(1)
	trans.MediaFile().SetSkipVideo(true)
	trans.MediaFile().SetAudioFilter("aresample=resampler=soxr")

	err := <-trans.Run(false) // 변환이 완료할 때까지 block
	if err != nil {
		return nil, fmt.Errorf("transcode audio file failed : %w", err)
	}

	file, err := os.Open(audioFileName8K)
	if err != nil {
		return nil, fmt.Errorf("open audio file failed: %w", err)
	}

	return &FileStreamer{file: file}, nil
}
func main() {
	// arg 설정. 하나의 오디오 파일을 flag로 받음
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <AUDIOFILE>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "<AUDIOFILE> must be a path to a local audio file. Audio file must be a 16-bit signed little-endian encoded with a sample rate of 16000.\n")
	}
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("Please pass path to your local audio file as a command line argument")
	}

	audioFile := flag.Arg(0)

	//JWT 토큰을 받기 위한 과정
	data := map[string][]string{
		"client_id":     {ClientId},
		"client_secret": {ClientSecret},
	}

	resp, _ := http.PostForm("https://openapi.vito.ai/v1/authenticate", data)
	if resp.StatusCode != 200 {
		panic("Failed to authenticate")
	}

	// 결과값 중에서 access_token 값만을 result 에 할당
	bytes, _ := io.ReadAll(resp.Body)
	var result struct {
		Token string `json:"access_token"`
	}
	json.Unmarshal(bytes, &result)

	//// 스트리밍 gRPC 예제
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))) //TLS certificate nil로 해서 header를 통한 authority 진행

	//Current way
	conn, err := grpc.NewClient(ServerHost, dialOpts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn.WaitForStateChange(ctx, connectivity.Ready)

	md := metadata.Pairs("authorization", fmt.Sprintf("%s %v", "bearer", result.Token))

	ctx = context.Background()
	newCtx := metautils.NiceMD(md).ToOutgoing(ctx) //metautils를 통해 요청 보낼때 header에 bearer 인증 반복적으로 보냄

	// 여기서부터 VITO gRPC 함수
	client := pb.NewOnlineDecoderClient(conn)

	stream, err := client.Decode(newCtx)

	if err != nil {
		log.Printf("Failed to create stream: %v\n", err)
		log.Fatal(err)
	}

	// Send the initial configuration message.
	if err := stream.Send(&pb.DecoderRequest{
		StreamingRequest: &pb.DecoderRequest_StreamingConfig{
			StreamingConfig: &pb.DecoderConfig{
				SampleRate:          int32(SAMPLE_RATE),
				Encoding:            pb.DecoderConfig_LINEAR16,
				UseItn:              &True,
				UseDisfluencyFilter: &False,
				UseProfanityFilter:  &False,
			},
		},
	}); err != nil {
		log.Fatal(err)
	}

	// AudioFile을 열기
	streamingFile, err := OpenAudioFile(audioFile)
	if err != nil {
		log.Fatal(err)
	}
	defer streamingFile.Close()

	//Wait Group을 걸어 안전하게 file을 다 읽고나서 종료하게끔 구현
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := streamingFile.Read(buf)
			if n > 0 {
				if err := stream.Send(&pb.DecoderRequest{
					StreamingRequest: &pb.DecoderRequest_AudioContent{
						AudioContent: buf[:n],
					},
				}); err != nil {
					log.Printf("Could not send audio: %v", err)
				}
			}
			if err == io.EOF {
				// Nothing else to pipe, close the stream.
				if err := stream.CloseSend(); err != nil {
					log.Fatalf("Could not close stream: %v", err)
				}
				return
			}
			if err != nil {
				log.Printf("Could not read from %s: %v", audioFile, err)
				continue
			}
		}
	}()

	_, err = stream.Recv()
	if err != nil {
		log.Fatalf("failed to recv: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Cannot stream results: %v", err)
			break
		}

		if err := resp.Error; err {
			log.Printf("Could not recognize: %v", err)
			break
		}
		for _, result := range resp.Results {
			if result.IsFinal {
				fmt.Printf("final: %v\n", result.Alternatives[0].Text)
			} else {
				fmt.Printf("%v\n", result.Alternatives[0].Text)
			}
		}
	}
	wg.Wait()
}
