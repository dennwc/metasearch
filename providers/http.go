package providers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ErrHTTPStatus struct {
	Code   int
	Status string
}

func (e *ErrHTTPStatus) Error() string {
	return fmt.Sprintf("status: %v", e.Status)
}

var debugHTTP = os.Getenv("METAS_DEBUG_HTTP") == "true"

func NewHTTPClient(base string) HTTPClient {
	return HTTPClient{
		cli:  http.DefaultClient,
		base: base,
	}
}

type HTTPClient struct {
	cli  *http.Client
	base string
}

func (c *HTTPClient) SetHTTPClient(cli *http.Client) {
	c.cli = cli
}

func (c *HTTPClient) url(path string) string {
	return c.base + path
}

func (c *HTTPClient) GetRequest(path string, params url.Values) (*http.Request, error) {
	addr := c.url(path)
	if len(params) != 0 {
		addr += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return nil, err
	}
	if debugHTTP {
		log.Println("GET", addr)
	}
	return req, nil
}

func (c *HTTPClient) postReq(path string, params url.Values) (*http.Request, error) {
	addr := c.url(path)
	var r io.Reader
	if len(params) != 0 {
		r = strings.NewReader(params.Encode())
	}
	req, err := http.NewRequest("POST", addr, r)
	if err != nil {
		return nil, err
	}
	if len(params) != 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if debugHTTP {
		log.Println("POST", addr, params)
	}
	return req, nil
}

func (c *HTTPClient) Get(ctx context.Context, path string, params url.Values) (*http.Response, error) {
	req, err := c.GetRequest(path, params)
	if err != nil {
		return nil, err
	}
	return c.DoRaw(ctx, req)
}

func (c *HTTPClient) DoRaw(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return c.cli.Do(req)
}

func (c *HTTPClient) doEnc(ctx context.Context, req *http.Request, accept string) (*http.Response, error) {
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.DoRaw(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		resp.Body.Close()
		return nil, &ErrHTTPStatus{Status: resp.Status, Code: resp.StatusCode}
	}
	return resp, nil
}

func (c *HTTPClient) getEnc(ctx context.Context, path string, params url.Values, accept string) (*http.Response, error) {
	req, err := c.GetRequest(path, params)
	if err != nil {
		return nil, err
	}
	return c.doEnc(ctx, req, accept)
}

func (c *HTTPClient) postEnc(ctx context.Context, path string, params url.Values, accept string) (*http.Response, error) {
	req, err := c.postReq(path, params)
	if err != nil {
		return nil, err
	}
	return c.doEnc(ctx, req, accept)
}

func (c *HTTPClient) GetJSON(ctx context.Context, path string, params url.Values, dst interface{}) error {
	resp, err := c.getEnc(ctx, path, params, "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body
	if debugHTTP {
		r = io.TeeReader(r, os.Stderr)
	}

	dec := json.NewDecoder(r)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func (c *HTTPClient) GetXML(ctx context.Context, path string, params url.Values, dst interface{}) error {
	resp, err := c.getEnc(ctx, path, params, "application/xml")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body
	if debugHTTP {
		r = io.TeeReader(r, os.Stderr)
	}

	dec := xml.NewDecoder(r)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func (c *HTTPClient) DoHTML(ctx context.Context, req *http.Request) (*goquery.Document, error) {
	resp, err := c.doEnc(ctx, req, "text/html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body
	if debugHTTP {
		r = io.TeeReader(r, os.Stderr)
	}

	return goquery.NewDocumentFromReader(r)
}

func (c *HTTPClient) GetHTML(ctx context.Context, path string, params url.Values) (*goquery.Document, error) {
	resp, err := c.getEnc(ctx, path, params, "text/html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body
	if debugHTTP {
		r = io.TeeReader(r, os.Stderr)
	}

	return goquery.NewDocumentFromReader(r)
}

func (c *HTTPClient) PostHTML(ctx context.Context, path string, params url.Values) (*goquery.Document, error) {
	resp, err := c.postEnc(ctx, path, params, "text/html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r io.Reader = resp.Body
	if debugHTTP {
		r = io.TeeReader(r, os.Stderr)
	}

	return goquery.NewDocumentFromReader(r)
}
