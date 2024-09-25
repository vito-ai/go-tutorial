# RTZR STT API Go example

RTZR Streaming STT API Example

## Authentication

* Sign up [ReturnZero Developer Web][rtzr-dev], and enter [MY Console][my-console]
* From the MY Console, create an application,
  copy its CLIENT_ID and CLIENT_SECRET, then set the 
 `ClientId` and `ClientSecret` variable in the file.

[rtzr-dev]: https://developers.rtzr.ai/
[my-console]: https://developers.rtzr.ai/dashboard

## Run the sample

Before running any example you must first install below:

```bash
go get -u github.com/vito-ai/go-genproto
go get -u google.golang.org/grpc
go get -u github.com/grpc-ecosystem/go-grpc-middleware
go get -u github.com/xfrr/goffmpeg
```

To run the example with a local file:

```bash
go run batchapi.go ../testdata/sample.wav
```