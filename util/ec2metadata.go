package util

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

type IEC2Metadata interface {
	GetEC2Metadata() (map[string]interface{}, error)
	GetEC2MetadataWithRetry(numberOfRetries int) (map[string]interface{}, error)
	GetEC2InstanceIDWithRetry(numberOfRetries int) (string, error)
}

type EC2MetadataClient struct {
	ec2MetadataURL           string
	imdSV2URL                string
	currentToken             *token
	tokenExpirationInSeconds int
}

type token struct {
	Token           string
	TokenExpiration time.Time
}

const ec2MetadataURL = "http://169.254.169.254/latest"
const imdSV2URL = "http://169.254.169.254/latest/api/token"

func New(tokenExpirationInSeconds int) *EC2MetadataClient {
	if tokenExpirationInSeconds < 1 {
		log.Fatal("tokenExpirationInSeconds must be greater than 0")
	}

	client := &EC2MetadataClient{
		ec2MetadataURL:           ec2MetadataURL,
		imdSV2URL:                imdSV2URL,
		tokenExpirationInSeconds: tokenExpirationInSeconds,
	}
	return client
}

func (client *EC2MetadataClient) setIMDSv2Token(numberOfRetries int) (err error) {

	if client.currentToken == nil || client.currentToken.TokenExpiration.Before(time.Now()) {
		var token *token
		for i := 0; i < numberOfRetries; i++ {
			token, err = client.get_IMD_sv2_SecurityToken(client.tokenExpirationInSeconds)
			if err == nil {
				client.currentToken = token
				return nil
			}
		}
		return err
	}

	return nil
}

func (client *EC2MetadataClient) GetEC2InstanceIDWithRetry(numberOfRetries int) (resp string, err error) {

	if numberOfRetries < 1 {
		log.Fatal("numberOfRetries must be greater than zero")
	}

	// Make sure token is not expired, renew if necessary
	//  Try IMDsv2 method first
	err = client.setIMDSv2Token(numberOfRetries)
	if err == nil {
		for i := 0; i < numberOfRetries; i++ {
			resp, err = client.getEC2InstanceID_using_IMD_sv2(client.currentToken.Token)
			if err == nil {
				return resp, nil
			}
		}

	}

	//  If IMDsv2 is not successful, then try IMDsv1

	// First, get the EC2 instance ID using IMDSv2.
	for n := 0; n < numberOfRetries; n++ {
		resp, err = client.getEC2InstanceID_using_IMD_sv1()
		if err == nil {
			return resp, nil
		}
	}
	return "", err
}

func (client *EC2MetadataClient) get_IMD_sv2_SecurityToken(tokenExpirationInSeconds int) (*token, error) {

	putRequest, err := http.NewRequest("PUT", client.imdSV2URL, nil)
	if err != nil {
		return nil, err
	}
	putRequest.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", strconv.Itoa(tokenExpirationInSeconds))

	//  Submit POST request
	var putResponse *http.Response
	putResponse, err = http.DefaultClient.Do(putRequest)
	if err != nil {
		return nil, err
	}

	defer putResponse.Body.Close()
	var body []byte
	body, err = ioutil.ReadAll(putResponse.Body)

	if err != nil {
		return nil, err
	}
	tokenResp := &token{Token: string(body), TokenExpiration: time.Now().Add(time.Second * time.Duration(tokenExpirationInSeconds))}
	return tokenResp, nil
}

func (client *EC2MetadataClient) getEC2InstanceID_using_IMD_sv2(token string) (string, error) {

	urlPath, err := url.Parse(client.ec2MetadataURL)
	if err != nil {
		return "", err
	}

	urlPath.Path = path.Join(urlPath.Path, "meta-data/instance-id")

	getRequest, err := http.NewRequest("GET", urlPath.String(), nil)
	if err != nil {
		return "", err
	}
	getRequest.Header.Add("X-aws-ec2-metadata-token", token)

	resp, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (client *EC2MetadataClient) getEC2InstanceID_using_IMD_sv1() (string, error) {
	urlPath, err := url.Parse(client.ec2MetadataURL)
	if err != nil {
		log.Fatal(err)
	}

	urlPath.Path = path.Join(urlPath.Path, "meta-data/instance-id")
	resp, err := http.Get(urlPath.String())
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (client *EC2MetadataClient) GetEC2MetadataWithRetry(numberOfRetries int) (map[string]interface{}, error) {
	var err error
	var response map[string]interface{}
	for i := 0; i < numberOfRetries; i++ {
		response, err = client.GetEC2Metadata()
		if err == nil {
			return response, nil
		}
	}
	return nil, err

}

func (client *EC2MetadataClient) GetEC2Metadata() (map[string]interface{}, error) {

	urlPath, err := url.Parse(client.ec2MetadataURL)
	if err != nil {
		log.Fatal(err)
	}

	urlPath.Path = path.Join(urlPath.Path, "dynamic/instance-identity/document")
	resp, err := http.Get(urlPath.String())
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(body), &jsonMap)

	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}
