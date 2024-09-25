# RTZR STT API Go example

RTZR Batch STT API Example

## Authentication

* Sign up [ReturnZero Developer Web][rtzr-dev], and enter [MY Console][my-console]
* From the MY Console, create an application,
  copy its CLIENT_ID and CLIENT_SECRET, then set the 
  `ClientId` and `ClientSecret` variable in the file.

[rtzr-dev]: https://developers.rtzr.ai/
[my-console]: https://developers.rtzr.ai/dashboard

## Run the sample

To run the example with a local file:

```bash
go run main.go ../testdata/sample.wav
```