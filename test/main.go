package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("testmain")

func main() {
	logging.SetLogLevel("*", "INFO")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli, err := network.NewIPLDClient(ctx, datastore.NewMapDatastore())
	panicErr(err)
	sw := &safewrap.SafeWrap{}
	n := sw.WrapObject(map[string]string{"hi": "world", "im": "tupelohere"})
	panicErr(sw.Err)
	fmt.Printf("putting %s\n", n.Cid().String())
	cli.Add(ctx, n)

	reader := bytes.NewReader(n.RawData())

	resp, err := newfileUploadRequest(
		"https://ipfs.infura.io:5001/api/v0/dag/put?format=cbor&input-enc=raw",
		nil, "file", reader)
	panicErr(err)

	log.Infof("resp %v", resp)

	<-make(chan struct{})
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, paramName string, file io.Reader) (*http.Response, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, "memory.cbor")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err != nil {
		return nil, errors.Wrap(err, "error making request")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	return resp, err
}
