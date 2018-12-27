package search

import (
	"net/url"
)

type LinkResult struct {
	URL   url.URL
	Title string
	Desc  string
}

func (r *LinkResult) GetURL() *url.URL {
	u := r.URL
	return &u
}

func (r *LinkResult) GetTitle() string {
	return r.Title
}

func (r *LinkResult) GetDesc() string {
	return r.Desc
}

var _ ThumbnailResult = (*ImageResult)(nil)

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

func (r *ImageResult) GetThumbnail() *Image {
	if r.Thumbnail != nil {
		return r.Thumbnail
	}
	return &r.Image
}

type Image struct {
	URL    url.URL
	Width  int
	Height int
}

func (r *Image) GetURL() *url.URL {
	u := r.URL
	return &u
}

var _ ThumbnailResult = (*VideoResult)(nil)

type VideoResult struct {
	LinkResult
	Thumbnail *Image
}

func (r *VideoResult) GetThumbnail() *Image {
	return r.Thumbnail
}

var _ ThumbnailResult = (*EntityResult)(nil)

type EntityResult struct {
	LinkResult
	Type     string
	Category string
	Image    *Image
}

func (r *EntityResult) GetThumbnail() *Image {
	return r.Image
}
