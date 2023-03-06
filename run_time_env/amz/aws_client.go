package amz

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

type tokenInfo struct {
	Value           string
	TokenExpiration time.Time
}

func (t *tokenInfo) IsValid() bool {
	return t.TokenExpiration.After(time.Now())
}
func NewToken(value string, expirationTime int) *tokenInfo {
	return &tokenInfo{Value: value, TokenExpiration: time.Now().Add(time.Second * time.Duration(expirationTime))}
}

const (
	tokenExpirationInSeconds = 6 * 3600
	numberOfRetries          = 3
)

type amz_client struct {
	Token *tokenInfo
	doc   *ec2metadata.EC2InstanceIdentityDocument // cached data
}

func NewClient() (client *amz_client, err error) {
	// on EC2 this may fail, but then nil token will work fine
	t, _ := retrieveSecurityToken(3, tokenExpirationInSeconds)

	c := amz_client{Token: t}
	doc, err := c.getInstanceIdentityDocument()
	if err == nil {
		c.doc = &doc
		client = &c
	}
	return
}

func (client *amz_client) getToken() (token string, err error) {

	if client.Token == nil {
		err = errors.New("token is not required")
		return
	}
	if !client.Token.IsValid() {
		var t *tokenInfo
		t, err = retrieveSecurityToken(3, tokenExpirationInSeconds)
		client.Token = t
	}
	if client.Token != nil {
		token = client.Token.Value
	}
	return
}

func (client *amz_client) GetMetadata(name string) (data string, err error) {
	resp, err := client.query(fmt.Sprintf(`meta-data/%v`, name))
	data = string(resp)
	return
}

func (client *amz_client) query(name string) (resp []byte, err error) {

	url := formatURL(name)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	token, err := client.getToken()
	if err == nil {
		request.Header.Add("X-aws-ec2-metadata-token", token)
	}
	resp, err = processRequest(request)
	return
}

func (client *amz_client) GetInstanceIdentityDocument() (doc ec2metadata.EC2InstanceIdentityDocument, err error) {
	if client.doc != nil {
		doc = *client.doc
	} else {
		doc, err = client.getInstanceIdentityDocument()
	}
	return
}

func (client *amz_client) getInstanceIdentityDocument() (doc ec2metadata.EC2InstanceIdentityDocument, err error) {
	resp, err := client.query("dynamic/instance-identity/document")

	if err == nil {
		err = json.Unmarshal(resp, &doc)
	}
	return
}

func processRequest(request *http.Request) (response []byte, err error) {

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		err = fmt.Errorf(`req={%v %v %v}  resp={%v %v}`, request.Method, request.URL, request.Header, resp.Status, resp.Header)
	} else {
		response = body
	}
	return
}

func retrieveSecurityTokenOnce(expirationTime int) (t *tokenInfo, err error) {
	request, err := http.NewRequest(http.MethodPut, formatURL("api/token"), nil)
	if err != nil {
		return
	}
	request.Header.Add("X-aws-ec2-metadata-token-ttl-seconds", strconv.Itoa(expirationTime))

	body, err := processRequest(request)
	if err == nil {
		t = NewToken(string(body), expirationTime)
	}
	return
}

func retrieveSecurityToken(numberOfRetries int, expirationTime int) (t *tokenInfo, err error) {
	for i := 0; i < numberOfRetries; i++ {
		t, err = retrieveSecurityTokenOnce(expirationTime)
		if err == nil {
			return
		}
	}
	return
}

func formatURL(relativePath string) string {
	return fmt.Sprintf(`http://169.254.169.254/latest/%v`, relativePath)
}

// type amz_ec2_client struct {
// 	ec2Metadata *ec2metadata.EC2Metadata
// }

// func NewEC2Client() (client *amz_ec2_client) {
// 	sess, err := session.NewSession()
// 	if err != nil {
// 		return
// 	}
// 	ec2Meta := ec2metadata.New(sess)
// 	return &amz_ec2_client{ec2Metadata: ec2Meta}
// }

// func (client *amz_ec2_client) GetMetadata(name string) (data string, err error) {
// 	return client.ec2Metadata.GetMetadata(name)
// }
// func (client *amz_ec2_client) GetInstanceIdentityDocument() (doc ec2metadata.EC2InstanceIdentityDocument, err error) {
// 	return client.ec2Metadata.GetInstanceIdentityDocument()
// }
