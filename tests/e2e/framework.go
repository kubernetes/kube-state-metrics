/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const (
	epHealthz = "/healthz"
	epMetrics = "/metrics"
)

var (
	framework *Framework
)

type Framework struct {
	KsmClient *KSMClient
}

func NewFramework(ksmurl string) (*Framework, error) {
	ksmClient, err := NewKSMClient(ksmurl)
	if err != nil {
		return nil, err
	}

	return &Framework{
		KsmClient: ksmClient,
	}, nil
}

type KSMClient struct {
	endpoint *url.URL
	client   *http.Client
}

func NewKSMClient(address string) (*KSMClient, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/")

	return &KSMClient{
		endpoint: u,
		client:   &http.Client{},
	}, nil
}

func (k *KSMClient) isHealthz() (bool, error) {
	p := path.Join(k.endpoint.Path, epHealthz)

	u := *k.endpoint
	u.Path = p

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return true, nil
}

func (k *KSMClient) metrics(w io.Writer) error {
	p := path.Join(k.endpoint.Path, epMetrics)

	u := *k.endpoint
	u.Path = p

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	io.Copy(w, resp.Body)

	return nil
}
