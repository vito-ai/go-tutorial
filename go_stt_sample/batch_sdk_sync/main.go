package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vito-ai/speech"
)

var True = true
var False = false

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <AUDIOFILE>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "<AUDIOFILE> must be a path to a local audio file. Audio file must be a 16-bit signed little-endian encoded with a sample rate of 16000.\n")
	}
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("Please pass path to your local audio file as a command line argument")
	}

	filePath := flag.Arg(0)
	ctx := context.Background()

	client, err := speech.NewRestClient(nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := client.Recognize(ctx, &speech.RecognizeRequest{
		Config: speech.RecognitionConfig{
			ModelName: "sommers",
			UseItn:    &True,
		},
		AudioSource: speech.RecognitionAudio{
			FilePath: filePath,
		},
	})
	if err != nil && err != speech.ErrNotFinish {
		fmt.Println(err)
		return
	}

	for _, utterance := range resp.Results.Utterances {
		fmt.Println(utterance.Msg)
	}
	if err != nil {
		log.Fatal(err)
	}

}
