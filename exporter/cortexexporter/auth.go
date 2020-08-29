package cortexexporter

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

// SigningRoundTripper is a Custom RoundTripper that performs AWS Sig V4
type SigningRoundTripper struct {
	transport http.RoundTripper
	signer    *v4.Signer
	service   string
	cfg       *aws.Config
	debug	  bool
}

// RoundTrip signs each outgoing request
func (si *SigningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBody, err := req.GetBody()
	if err != nil {
		return nil, err
	}

	// Get the body
	content, err := ioutil.ReadAll(reqBody)
	if err != nil {
		return nil, err
	}

	body := bytes.NewReader(content)

	// Sign the request
	_, err = si.signer.Sign(req, body, si.service, *si.cfg.Region, time.Now())
	if err != nil {
		return nil, err
	}
	if si.debug {
		log.Printf("%+v\n",req)
	}
	// Send the request to Cortex
	resp, err := si.transport.RoundTrip(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if si.debug {
		log.Printf("%+v\n",resp)
	}

	return resp, err
}

// NewAuth takes a map of strings as parameters and return a http.RoundTripper
func NewAuth(params map[string]interface{}) (http.RoundTripper, error) {
	debug, found := params[debugStr]
	var debugFlag bool
	if found {
		debugFlag = debug == enabledStr
	}
	reg, found := params[regionStr]
	if !found {
		return nil, errors.New("plugin error: region not specified")
	}
	region, isString := reg.(string)
	if !isString {
		return nil, errors.New("plugin error: region is not string")
	}
	serv, found := params[serviceStr]
	if !found {
		return nil, errors.New("plugin error: service not specified")
	}

	service, isString := serv.(string)
	if !isString {
		return nil, errors.New("plugin error: region is not string")
	}

	client, found := params[origClientStr]
	if !found {
		return nil, errors.New("plugin error: default client not specified")
	}
	origClient, isClient := client.(*http.Client)
	if !isClient {
		return nil, errors.New("plugin error: default client not specified")
	}

	// Initialize session with default credential chain
	// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
		aws.NewConfig().WithLogLevel(aws.LogDebugWithSigning),
	)
	if err != nil {
		log.Println("AWS session initialization failed")
	}

	if _, err = sess.Config.Credentials.Get(); err != nil {
		log.Println("AWS session initialized, but credentials are not loaded correctly")
	}

	// Get Credentials, either from ./aws or from environmental variables
	creds := sess.Config.Credentials
	signer := v4.NewSigner(creds)
	if debugFlag {
		signer.Debug = aws.LogDebugWithSigning
		signer.Logger = aws.NewDefaultLogger()
	}
	rtp := SigningRoundTripper{
		transport: origClient.Transport,
		debug:	   debugFlag,
		signer:    signer,
		cfg:       sess.Config,
		service:   service,
	}
	// return a RoundTripper
	return &rtp, nil
}
