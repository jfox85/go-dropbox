package dropbox

import (
	"encoding/json"
	"io"
	"strings"
	"net/url"
	"fmt"
	"io/ioutil"
	"net/http"
)

//
// putUrl signs API PUT requests with oauth credentials
//

func (drop *DropboxClient) putUrl(putUrl string, params url.Values, body string) error {
	_, err := drop.fetchUrl("PUT", putUrl, params, body)
	return err
}

//
// postUrl signs API POST requests with oauth credentials
//

func (drop *DropboxClient) postUrl(postUrl string, params url.Values, body string) (*http.Response, error) {
	return drop.fetchUrl("POST", postUrl, params, body)
}

//
// fetchUrl signs API requests with oauth credentials
//

func (drop *DropboxClient) fetchUrl(method, reqUrl string, params url.Values, body string) (*http.Response, error) {
	if params == nil {
		params = make(url.Values)
	}

	client := &http.Client{}

	oauthClient.SignParam(drop.Creds, method, reqUrl, params)

	req, err := http.NewRequest(method, reqUrl+"?"+params.Encode(), strings.NewReader(body))
	resp, err := client.Do(req)

	return resp, err
}

//
// postUrlDecode makes an API request and json decodes the response into data
//

func (drop *DropboxClient) postUrlDecode(url string, params url.Values, body string, data interface{}) error {
	resp, err := drop.postUrl(url, params, body)
	if err != nil {
		fmt.Printf("error in postUrlDecode: %v", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("postUrlDecode request for %s returned %d, %s", url, resp.StatusCode, string(b))
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return err
	}

	return nil
}

//
// postUrlDecodeFile makes an API request and json decodes the response into a DropFile
//

func (drop *DropboxClient) postUrlDecodeFile(url string, params url.Values, body string) (*DropFile, error) {
	data := new(DropFile)

	err := drop.postUrlDecode(url, params, body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

//
// postUrlToReader makes an API request and returns a Reader on the response
//

func (drop *DropboxClient) postUrlToReader(url string, params url.Values, body string) (io.ReadCloser, error) {
	resp, err := drop.postUrl(url, params, body)
	if err != nil {
		fmt.Printf("error in postUrlToReader: %v", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("postUrlToReader request for %s returned %d, %s", url, resp.StatusCode, string(b))
	}

	return resp.Body, nil
}

//
// getUrl signs our API GET requests with our oauth credentials
//

func (drop *DropboxClient) getUrl(getUrl string, params url.Values, data interface{}) error {
	if params == nil {
		params = make(url.Values)
	}

	oauthClient.SignParam(drop.Creds, "GET", getUrl, params)
	res, err := http.Get(getUrl + "?" + params.Encode())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		b, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("Get request for %s returned %d, %s", getUrl, res.StatusCode, string(b))
	}

	return json.NewDecoder(res.Body).Decode(&data)
}

//
// apiContentURL is a path constructor function to build the proper URL for API_FILES requests
//

func apiContentURL(path string) string {
	fullurl, err := url.Parse(_API_FILES_URL + strings.TrimLeft(path, "/"))
	if err != nil {
		fmt.Printf("url parse error: %v", err)
	}

	return fullurl.String()
}
