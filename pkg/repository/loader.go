// Copyright © 2019 Ettore Di Giacinto <mudler@gentoo.org>
//                  Daniele Rondina <geaaru@sabayonlinux.org>
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

package repository

import (
	"io/ioutil"
	"path"
	"regexp"

	"github.com/ghodss/yaml"

	. "github.com/mudler/luet/pkg/config"
	. "github.com/mudler/luet/pkg/logger"
)

func LoadRepositories(c *LuetConfig) error {
	var regexRepo = regexp.MustCompile(`.yml$`)

	for _, rdir := range c.RepositoriesConfDir {
		Debug("Parsing Repository Directory", rdir, "...")

		files, err := ioutil.ReadDir(rdir)
		if err != nil {
			Warning("Skip dir", rdir, ":", err.Error())
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !regexRepo.MatchString(file.Name()) {
				Debug("File", file.Name(), "skipped.")
				continue
			}

			content, err := ioutil.ReadFile(path.Join(rdir, file.Name()))
			if err != nil {
				Warning("On read file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			r, err := LoadRepository(content)
			if err != nil {
				Warning("On parse file", file.Name(), ":", err.Error())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			if r.Name == "" || len(r.Urls) == 0 || r.Type == "" {
				Warning("Invalid repository ", file.Name())
				Warning("File", file.Name(), "skipped.")
				continue
			}

			c.AddSystemRepository(*r)
		}
	}
	return nil
}

func LoadRepository(data []byte) (*LuetRepository, error) {
	ans := NewEmptyLuetRepository()
	err := yaml.Unmarshal(data, &ans)
	if err != nil {
		return nil, err
	}
	return ans, nil
}
