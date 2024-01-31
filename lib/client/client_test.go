// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

type errorServer struct {
	t *testing.T
}

func (s errorServer) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 500,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`error`)),
		Header:     make(http.Header),
	}, nil
}

func TestError(t *testing.T) {
	var result jsonResult
	c := NewTransportGetter(Config{UserAgent: "test/1.0"}, errorServer{t: t})
	err := c.GetJson("https://host/path", &result)
	// expect retry backoff attempts
	if err == nil {
		t.Error("expect error")
	}
}

type jsonServer struct {
	t *testing.T
}

type jsonResult struct {
	A string `json:"a"`
}

func (s jsonServer) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"a":"b"}`)),
		Header:     make(http.Header),
	}, nil
}

func TestGetJson(t *testing.T) {
	// urls := []string{
	// 	"http://musicbrainz.org/ws/2/artist/5b11f4ce-a62d-471e-81fc-a69a8278c7da?inc=aliases&fmt=json",
	// 	"http://musicbrainz.org/ws/2/artist/5b11f4ce-a62d-471e-81fc-a69a8278c7da?inc=aliases&fmt=json",
	// 	"http://musicbrainz.org/ws/2/artist/5b11f4ce-a62d-471e-81fc-a69a8278c7da?inc=aliases&fmt=json",
	// 	"http://musicbrainz.org/ws/2/artist/ba0d6274-db14-4ef5-b28d-657ebde1a396?inc=aliases&fmt=json",
	// 	"http://musicbrainz.org/ws/2/artist/ba0d6274-db14-4ef5-b28d-657ebde1a396?inc=aliases&fmt=json",
	// 	"http://musicbrainz.org/ws/2/artist/ba0d6274-db14-4ef5-b28d-657ebde1a396?inc=aliases&fmt=json",
	// }

	var result jsonResult
	c := NewTransportGetter(Config{UserAgent: "test/1.0"}, jsonServer{t: t})
	err := c.GetJson("https://host/path", &result)
	if err != nil {
		t.Error(err)
	}
	if result.A != "b" {
		t.Errorf("expect 'b' got '%s'", result.A)
	}
}

type xmlServer struct {
	t *testing.T
}

type xmlResult struct {
	Flag  bool   `xml:"flag,attr"`
	Value string `xml:",chardata"`
}

func (s xmlServer) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`<a flag="true">b</a>`)),
		Header:     make(http.Header),
	}, nil
}

func TestGetXML(t *testing.T) {
	var result xmlResult
	c := NewTransportGetter(Config{UserAgent: "test/1.0"}, xmlServer{t: t})
	err := c.GetXML("https://host/path", &result)
	if err != nil {
		t.Error(err)
	}
	if result.Flag != true {
		t.Errorf("expect flag true")
	}
	if result.Value != "b" {
		t.Errorf("expect 'b' got '%s'", result.Value)
	}
}

type plsServer struct {
	t *testing.T
}

func (s plsServer) RoundTrip(r *http.Request) (*http.Response, error) {
	body := `
[playlist]
numberofentries=1
File1=https://ice6.somafm.com/brfm-128-mp3
Title1=SomaFM: Black Rock FM (#1): From the Playa to the world, for the annual Burning Man festival.
Length1=-1
`
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

func TestGetPLS(t *testing.T) {
	c := NewTransportGetter(Config{UserAgent: "test/1.0"}, plsServer{t: t})
	result, err := c.GetPLS("https://host/path")
	if err != nil {
		t.Error(err)
	}
	if result.NumberOfEntries != 1 {
		t.Error("expect 1 entry")
	}
	if len(result.Entries) != 1 {
		t.Error("expect len 1")
	}
	if result.Entries[0].File != "https://ice6.somafm.com/brfm-128-mp3" {
		t.Error("expect mp3")
	}
	if result.Entries[0].Title !=
		"SomaFM: Black Rock FM (#1): From the Playa to the world, for the annual Burning Man festival." {
		t.Error("expect black rock fm")
	}
	if result.Entries[0].Length != -1 {
		t.Error("expect length -1")
	}
}
