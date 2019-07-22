package goroon

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"
)

const (
	packageVerSessionKey = "CBSESSID"
	cloudVerSessionKey   = "JSESSIONID"
)

var cloudURLPattern = regexp.MustCompile("[a-z]+.cybozu.com")

type Client struct {
	Endpoint   string
	Username   string
	Password   string
	Locale     string
	Debugger   io.Writer
	SessionKey string
	SessionId  string
}

type NopWriter struct{}

func (d *NopWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func NewClient(endpoint string) *Client {
	sessionKey := packageVerSessionKey
	if cloudURLPattern.MatchString(endpoint) {
		sessionKey = cloudVerSessionKey
	}

	return &Client{
		Endpoint:   endpoint,
		Locale:     "ja",
		Debugger:   &NopWriter{},
		SessionKey: sessionKey,
	}
}

func (c *Client) Request(action string, path string, req interface{}, res interface{}) error {
	envelope := &SoapEnvelope{}
	envelope.SoapHeader = c.createSoapHeader(action)
	envelope.SoapBody = &SoapBody{Content: req}
	b, err := xml.MarshalIndent(envelope, "", "	")
	c.Debugger.Write(b)
	msg, err := xml.Marshal(envelope)
	if err != nil {
		return err
	}
	client := c.createHttpClient()
	buf := bytes.NewBuffer(msg)
	resp, err := client.Post(c.Endpoint+path, "text/xml", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	c.Debugger.Write(body)

	res_env := &SoapEnvelope{SoapBody: &SoapBody{Content: res}}
	if err = xml.Unmarshal(body, res_env); err != nil {
		return err
	}
	if res_env.SoapBody.Fault != nil {
		msg := fmt.Sprintf("Soap Fault is occured: %s: %s", res_env.SoapBody.Fault.Detail.Diagnosis, res_env.SoapBody.Fault.Detail.Cause)
		return errors.New(msg)
	}
	return nil
}

func (c *Client) ScheduleGetEventsByTarget(params *Parameters) (*Returns, error) {
	req := &ScheduleGetEventsByTargetRequest{
		Parameters: params,
	}
	res := &ScheduleGetEventsByTargetResponse{}
	if err := c.Request("ScheduleGetEventsByTarget", "/cbpapi/schedule/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) UtilGetLoginUserId(params *Parameters) (*Returns, error) {
	req := &UtilGetLoginUserIdRequest{
		Parameters: params,
	}
	res := &UtilGetLoginUserIdResponse{}
	if err := c.Request("UtilGetLoginUserId", "/cbpapi/util/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) ScheduleGetEvents(params *Parameters) (*Returns, error) {
	req := &ScheduleGetEventsRequest{
		Parameters: params,
	}
	res := &ScheduleGetEventsResponse{}
	if err := c.Request("ScheduleGetEvents", "/cbpapi/schedule/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) BaseGetUsersByLoginName(params *Parameters) (*Returns, error) {
	req := &BaseGetUsersByLoginNameRequest{
		Parameters: params,
	}
	res := &BaseGetUsersByLoginNameResponse{}
	if err := c.Request("BaseGetUsersByLoginName", "/cbpapi/base/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) BulletinGetFollows(params *Parameters) (*Returns, error) {
	req := &BulletinGetFollowsRequest{
		Parameters: params,
	}
	res := &BulletinGetFollowsResponse{}
	if err := c.Request("BulletinGetFollows", "/cbpapi/bulletin/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) UtilLogin(params *Parameters) (*Returns, error) {
	req := &UtilLoginRequest{
		Parameters: params,
	}
	res := &UtilLoginResponse{}
	if err := c.Request("UtilLogin", "/util_api/util/api", req, res); err != nil {
		return nil, err
	}
	return res.Returns, nil
}

func (c *Client) createSoapHeader(action string) *SoapHeader {
	header := &SoapHeader{
		Locale:    c.Locale,
		Timestamp: Timestamp{},
	}
	header.Action = action
	created := time.Now()
	header.Timestamp.Created = created
	expires := created.Add(time.Duration(1) * time.Hour)
	header.Timestamp.Expires = expires

	if c.Username != "" {
		header.Security = Security{
			UsernameToken: UsernameToken{
				Username: c.Username,
				Password: c.Password,
			},
		}
	}
	return header
}

func (c *Client) createHttpClient() *http.Client {
	client := &http.Client{}
	if c.SessionId != "" {
		u, _ := url.Parse(c.Endpoint)
		cookie := &http.Cookie{
			Name:   c.SessionKey,
			Value:  c.SessionId,
			Path:   "/",
			Domain: u.Host,
		}
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(u, []*http.Cookie{cookie})
		client.Jar = jar

	}
	return client
}
