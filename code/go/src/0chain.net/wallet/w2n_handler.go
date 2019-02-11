package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"0chain.net/datastore"
)

type SendHandler func(uri string) bool
type EntitySendHandler func(entity datastore.Entity) SendHandler
type FormSendHandler func(map[string]string, interface{}) SendHandler

var transport *http.Transport
var httpClient *http.Client

func init() {
	transport = &http.Transport{MaxIdleConnsPerHost: 5}
	httpClient = &http.Client{Transport: transport, Timeout: time.Millisecond * 2000}
}

func SendEntityHandler(uri string) EntitySendHandler {
	return func(entity datastore.Entity) SendHandler {
		return func(receiver string) bool {
			var buffer *bytes.Buffer
			buffer = datastore.ToJSON(entity)
			url := fmt.Sprintf("%v%v", receiver, uri)
			req, err := http.NewRequest("POST", url, buffer)
			if err != nil {
				return false
			}
			defer req.Body.Close()
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			resp, err := httpClient.Do(req)
			if err != nil {
				return false
			}
			var rbuf bytes.Buffer
			rbuf.ReadFrom(resp.Body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return false
			}
			io.Copy(ioutil.Discard, resp.Body)
			return true
		}
	}
}

func SendFormHandler(uri string) FormSendHandler {
	return func(values map[string]string, thing interface{}) SendHandler {
		return func(receiver string) bool {
			form := url.Values{}
			url0 := fmt.Sprintf("%v%v", receiver, uri)
			for key, value := range values {
				form.Add(key, value)
			}
			resp, err := httpClient.PostForm(url0, form)
			if err != nil {
				return false
			}
			var rbuf bytes.Buffer
			rbuf.ReadFrom(resp.Body)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				var rbuf bytes.Buffer
				rbuf.ReadFrom(resp.Body)
				return false
			}
			if thing != nil {
				err := json.Unmarshal(rbuf.Bytes(), &thing)
				if err != nil {
					return false
				}
			}
			io.Copy(ioutil.Discard, resp.Body)
			return true
		}
	}
}

func (nodes Nodes) SendAtleast(numNodes int, handler SendHandler) Nodes {
	return nodes.sendTo(numNodes, handler)
}

func (nodes Nodes) sendTo(numNodes int, handler SendHandler) Nodes {
	var sendTo Nodes
	validCount := 0
	sendBucket := make(chan string, numNodes)
	validBucket := make(chan string, numNodes)
	done := make(chan bool, numNodes)
	for i := 0; i < numNodes; i++ {
		go func() {
			for node := range sendBucket {
				valid := handler(node)
				if valid {
					validBucket <- node
				}
				done <- true
			}
		}()
	}
	for _, node := range nodes {
		sendBucket <- node
	}
	doneCount := 0
	sendTimeout := time.NewTimer(time.Millisecond * 500)
	for true {
		select {
		case node := <-validBucket:
			sendTo = append(sendTo, node)
			validCount++
			if validCount == numNodes {
				close(sendBucket)
				return sendTo
			}
		case <-done:
			doneCount++
			if doneCount >= numNodes {
				close(sendBucket)
				return sendTo
			}
		case <-sendTimeout.C:
			close(sendBucket)
			return sendTo
		}
	}
	return sendTo
}
