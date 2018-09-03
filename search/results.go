package search

import (
	"net/url"
)

type LinkResult struct {
	URL   url.URL
	Title string
	Desc  string
}

func (r *LinkResult) GetURL() url.URL {
	return r.URL
}

func (r *LinkResult) GetTitle() string {
	return r.Title
}

func (r *LinkResult) GetDesc() string {
	return r.Desc
}

type ImageResult struct {
	Image
	Title     string
	Desc      string
	PageURL   *url.URL
	Thumbnail *Image
}

func (r *ImageResult) GetTitle() string {
	return r.Title
}

func (r *ImageResult) GetDesc() string {
	return r.Desc
}

type Image struct {
	URL    url.URL
	Width  int
	Height int
}

func (r *Image) GetURL() url.URL {
	return r.URL
}

type VideoResult struct {
	LinkResult
	Thumbnail *Image
}

type EntityResult struct {
	LinkResult
	Type     string
	Category string
	Image    *Image
}
