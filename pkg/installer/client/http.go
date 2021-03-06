// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package client

import (
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"path/filepath"

	. "github.com/mudler/luet/pkg/logger"

	"github.com/mudler/luet/pkg/compiler"
	"github.com/mudler/luet/pkg/config"
	"github.com/mudler/luet/pkg/helpers"

	"github.com/cavaliercoder/grab"
)

type HttpClient struct {
	RepoData RepoData
}

func NewHttpClient(r RepoData) *HttpClient {
	return &HttpClient{RepoData: r}
}

func (c *HttpClient) PrepareReq(dst, url string) (*grab.Request, error) {

	req, err := grab.NewRequest(dst, url)
	if err != nil {
		return nil, err
	}

	if val, ok := c.RepoData.Authentication["token"]; ok {
		req.HTTPRequest.Header.Set("Authorization", "token "+val)
	} else if val, ok := c.RepoData.Authentication["basic"]; ok {
		req.HTTPRequest.Header.Set("Authorization", "Basic "+val)
	}

	return req, err
}

func Round(input float64) float64 {
	if input < 0 {
		return math.Ceil(input - 0.5)
	}
	return math.Floor(input + 0.5)
}

func (c *HttpClient) DownloadArtifact(artifact compiler.Artifact) (compiler.Artifact, error) {
	var u *url.URL = nil
	var err error
	var req *grab.Request
	var temp string

	artifactName := path.Base(artifact.GetPath())
	cacheFile := filepath.Join(config.LuetCfg.GetSystem().GetSystemPkgsCacheDirPath(), artifactName)
	ok := false

	// Check if file is already in cache
	if helpers.Exists(cacheFile) {
		Info("Use artifact", artifactName, "from cache.")
	} else {

		temp, err = ioutil.TempDir(os.TempDir(), "tree")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(temp)

		client := grab.NewClient()

		for _, uri := range c.RepoData.Urls {
			Info("Downloading artifact", artifactName, "from", uri)

			u, err = url.Parse(uri)
			if err != nil {
				continue
			}
			u.Path = path.Join(u.Path, artifactName)

			req, err = c.PrepareReq(temp, u.String())
			if err != nil {
				continue
			}

			resp := client.Do(req)
			if err = resp.Err(); err != nil {
				continue
			}

			Info("Downloaded", artifactName, "of",
				fmt.Sprintf("%.2f", (float64(resp.BytesComplete())/1000)/1000), "MB (",
				fmt.Sprintf("%.2f", (float64(resp.BytesPerSecond())/1024)/1024), "MiB/s )")

			Debug("Copying file ", filepath.Join(temp, artifactName), "to", cacheFile)
			err = helpers.CopyFile(filepath.Join(temp, artifactName), cacheFile)
			if err != nil {
				continue
			}

			ok = true
			break
		}

		if !ok {
			return nil, err
		}
	}

	newart := artifact
	newart.SetPath(cacheFile)
	return newart, nil
}

func (c *HttpClient) DownloadFile(name string) (string, error) {
	var file *os.File = nil
	var u *url.URL = nil
	var err error
	var req *grab.Request
	var temp string

	ok := false

	temp, err = ioutil.TempDir(os.TempDir(), "tree")
	if err != nil {
		return "", err
	}

	client := grab.NewClient()

	for _, uri := range c.RepoData.Urls {

		file, err = ioutil.TempFile(os.TempDir(), "HttpClient")
		if err != nil {
			continue
		}

		u, err = url.Parse(uri)
		if err != nil {
			continue
		}
		u.Path = path.Join(u.Path, name)

		Info("Downloading", u.String())

		req, err = c.PrepareReq(temp, u.String())
		if err != nil {
			continue
		}

		resp := client.Do(req)
		if err = resp.Err(); err != nil {
			continue
		}

		Info("Downloaded", filepath.Base(resp.Filename), "of",
			fmt.Sprintf("%.2f", (float64(resp.BytesComplete())/1000)/1000), "MB (",
			fmt.Sprintf("%.2f", (float64(resp.BytesPerSecond())/1024)/1024), "MiB/s )")

		err = helpers.CopyFile(filepath.Join(temp, name), file.Name())
		if err != nil {
			continue
		}
		ok = true
		break
	}

	if !ok {
		return "", err
	}

	return file.Name(), err
}
