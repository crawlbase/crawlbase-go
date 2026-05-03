package crawlbase

import (
	"context"
	"encoding/base64"
)

// ScreenshotsAPI captures a screenshot of any URL. Returns the image
// bytes; use ImageBytes() on the response to decode the base64 payload
// into raw bytes you can write to a file.
//
// See https://crawlbase.com/docs/screenshots-api for parameters
// (full-page, viewport, mobile, format, etc.).
type ScreenshotsAPI struct {
	*baseClient
}

// NewScreenshotsAPI constructs a Screenshots API client with the given
// token. Returns ErrTokenRequired on an empty token.
func NewScreenshotsAPI(token string) (*ScreenshotsAPI, error) {
	bc, err := newBaseClient(token, "screenshots")
	if err != nil {
		return nil, err
	}
	return &ScreenshotsAPI{baseClient: bc}, nil
}

// Get captures a screenshot of targetURL. The image is returned as a
// base64-encoded string in Response.Body — call ImageBytes(res) to get
// the raw image bytes ready for ioutil.WriteFile / image.Decode / etc.
func (a *ScreenshotsAPI) Get(targetURL string, options map[string]string) (*Response, error) {
	return a.GetWithContext(context.Background(), targetURL, options)
}

// GetWithContext is Get with cancellation / deadline / trace propagation.
func (a *ScreenshotsAPI) GetWithContext(ctx context.Context, targetURL string, options map[string]string) (*Response, error) {
	return a.request(ctx, "GET", targetURL, options, nil, "")
}

// ImageBytes decodes the base64-encoded screenshot in res.Body into raw
// image bytes. Returns an error if the body isn't valid base64 — this
// shouldn't happen on a successful screenshot response, but check
// res.StatusCode and res.PCStatus before calling.
func ImageBytes(res *Response) ([]byte, error) {
	return base64.StdEncoding.DecodeString(res.Body)
}
