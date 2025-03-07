# RTZR STT API Go example

RTZR Streaming STT SDK Example

## Authentication

* Sign up [ReturnZero Developer Web][rtzr-dev], and enter [MY Console][my-console]
* From the MY Console, create an application,
  copy its CLIENT_ID and CLIENT_SECRET, then set the 
  `CLIENT_ID` and `CLIENT_SECRET` environment variables:

  ```bash
  export CLIENT_ID=YOUR_CLIENT_ID
  export CLIENT_SECRET=YOUR_CLIENT_SECRET
  ```

[rtzr-dev]: https://developers.rtzr.ai/
[my-console]: https://developers.rtzr.ai/dashboard

## Run the sample

Before running any example you must first install below:

```bash
go get -u github.com/vito-ai/speech
go get -u github.com/vito-ai/go-genproto
go get -u github.com/xfrr/goffmpeg
```

To run the example with a local file:

```bash
go run main.go ../testdata/sample.wav
```