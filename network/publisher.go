package network

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
)

func newPublisherProps() *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &publisher{}
	})
}

type publisher struct{}

func (p *publisher) Receive(aCtx actor.Context) {
	switch msg := aCtx.Message().(type) {
	case *cbornode.Node:
		publishNode(msg)
	}
}

func publishNode(node *cbornode.Node) {
	reader := bytes.NewReader(node.RawData())

	resp, err := newfileUploadRequest(
		"https://ipfs.infura.io:5001/api/v0/dag/put?format=cbor&input-enc=raw",
		nil, "file", reader)
	log.Debugf("infura: (err: %v) %v", err, resp)

	if resp.StatusCode == 429 {
		time.Sleep(500 * time.Millisecond)
		publishNode(node)
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
	if err != nil {
		return nil, err
	}

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
