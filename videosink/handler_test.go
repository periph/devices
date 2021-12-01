// Copyright 2021 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package videosink

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type testCase struct {
	name          string
	opt           Options
	target        string
	wantMediaType string

	onImage func(*testing.T, image.Image)
}

func (tc *testCase) validatePart(t *testing.T, part *multipart.Part) {
	t.Helper()

	contentLength, err := strconv.ParseInt(part.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		t.Errorf("Parsing Content-Length header failed: %v", err)
	}

	decodeFunc := func(io.Reader) (image.Image, error) {
		return nil, errors.New("unknown image format")
	}

	if mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type")); err != nil {
		t.Errorf("ParseMediaType() failed: %v", err)
	} else if mediaType != tc.wantMediaType {
		t.Errorf("Got content-type %q, want %q", mediaType, tc.wantMediaType)
	} else {
		switch mediaType {
		case "image/png":
			decodeFunc = png.Decode
		case "image/jpeg":
			decodeFunc = jpeg.Decode
		}
	}

	if content, err := ioutil.ReadAll(part); err != nil {
		t.Errorf("ReadAll() failed: %v", err)
	} else if got, want := len(content), int(contentLength); got != want {
		t.Errorf("Read %d bytes, Content-Length header is %d", got, want)
	} else if img, err := decodeFunc(bytes.NewReader(content)); err != nil {
		t.Errorf("Decoding image failed: %v", err)
	} else if got, want := img.Bounds().Size(), (image.Point{tc.opt.Width, tc.opt.Height}); got != want {
		t.Errorf("Got image size %v, want %v", got, want)
	} else if tc.onImage != nil {
		tc.onImage(t, img)
	}

	if err := part.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func (tc *testCase) validateResponse(t *testing.T, resp *http.Response) {
	t.Helper()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("ServeHTTP() status %d, want %d", got, want)
	}

	if mediaType, mediaParams, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil {
		t.Errorf("ParseMediaType() failed: %v", err)
	} else if got, want := mediaType, "multipart/x-mixed-replace"; got != want {
		t.Errorf("Content-Type is %q, want %q", got, want)
	} else if boundary, ok := mediaParams["boundary"]; !(ok && len(boundary) > 50) {
		t.Errorf("Insufficient boundary: %s", boundary)
	} else {
		mr := multipart.NewReader(resp.Body, boundary)

		for {
			if part, err := mr.NextPart(); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Errorf("NextPart() failed: %v", err)
			} else {
				tc.validatePart(t, part)
			}
		}

		if _, err := mr.NextPart(); !(errors.Is(err, io.EOF) || strings.HasSuffix(err.Error(), " EOF")) {
			t.Errorf("Reading beyond last part didn't fail with EOF: %v", err)
		}
	}
}

func TestMultipartResponse(t *testing.T) {
	for _, tc := range []testCase{
		{
			name: "defaults",
			opt: Options{
				Width:  120,
				Height: 200,
				Format: DefaultFormat,
			},
			target:        "/",
			wantMediaType: "image/png",
		},
		{
			name: "default PNG",
			opt: Options{
				Width:  4,
				Height: 4,
				Format: PNG,
			},
			target:        "/",
			wantMediaType: "image/png",
		},
		{
			name: "default JPEG",
			opt: Options{
				Width:  200,
				Height: 100,
				Format: JPEG,
			},
			target:        "/",
			wantMediaType: "image/jpeg",
		},
		{
			name: "format param PNG",
			opt: Options{
				Width:  234,
				Height: 123,
				Format: JPEG,
			},
			target:        "/?format=png",
			wantMediaType: "image/png",
		},
		{
			name: "format param JPEG",
			opt: Options{
				Width:  123,
				Height: 456,
				Format: PNG,
			},
			target:        "/?format=jpeg",
			wantMediaType: "image/jpeg",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			d := New(&tc.opt)

			srv := httptest.NewServer(d)
			t.Cleanup(srv.Close)
			t.Cleanup(srv.CloseClientConnections)

			quit := make(chan struct{})
			remaining := 10

			tc.onImage = func(*testing.T, image.Image) {
				if remaining == 0 {
					tc.onImage = nil

					defer close(quit)

					if err := d.Halt(); err != nil {
						t.Errorf("Halt() failed: %v", err)
					}
				} else {
					remaining--
				}
			}

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					if err := d.Draw(d.Bounds(), image.Black, image.Point{}); err != nil {
						t.Errorf("Draw() failed: %v", err)
					}

					select {
					case <-quit:
						return
					case <-ctx.Done():
						return
					default:
					}

					time.Sleep(10 * time.Millisecond)
				}
			}()

			if resp, err := srv.Client().Get(srv.URL + tc.target); err != nil {
				t.Errorf("Get() failed: %v", err)
			} else {
				tc.validateResponse(t, resp)
			}

			if t.Failed() {
				cancel()
			}

			wg.Wait()
		})
	}
}

func TestRequestStatus(t *testing.T) {
	for _, tc := range []struct {
		method     string
		target     string
		wantStatus int
	}{
		{
			target:     "/?format=",
			wantStatus: http.StatusOK,
		},
		{
			target:     "/?format=bmp",
			wantStatus: http.StatusBadRequest,
		},
		{
			method:     http.MethodPost,
			target:     "/",
			wantStatus: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			d := New(&Options{
				Width:  16,
				Height: 16,
			})

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			t.Cleanup(cancel)

			srv := httptest.NewServer(d)
			t.Cleanup(srv.Close)
			t.Cleanup(srv.CloseClientConnections)

			req, err := http.NewRequestWithContext(ctx, tc.method, srv.URL+tc.target, nil)
			if err != nil {
				t.Errorf("NewRequest() failed: %v", err)
			}

			if resp, err := srv.Client().Do(req); err != nil {
				t.Errorf("Get() failed: %v", err)
			} else if got, want := resp.StatusCode, tc.wantStatus; got != want {
				t.Errorf("Request for %s %s returned status %d (%s), want %d",
					req.Method, req.URL.String(), got, resp.Status, want)
			}
		})
	}
}
