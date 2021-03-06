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

package helpers

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/archive"
)

func Tar(src, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	fs, err := archive.Tar(src, archive.Uncompressed)
	if err != nil {
		return err
	}
	defer fs.Close()

	_, err = io.Copy(out, fs)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}
	return err
}

// Untar just a wrapper around the docker functions
func Untar(src, dest string, sameOwner bool) error {
	var ans error

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if sameOwner {
		// PRE: i have root privileged.

		opts := &archive.TarOptions{
			// NOTE: NoLchown boolean is used for chmod of the symlink
			// Probably it's needed set this always to true.
			NoLchown:        true,
			ExcludePatterns: []string{"dev/"}, // prevent 'operation not permitted'
		}

		ans = archive.Untar(in, dest, opts)
	} else {

		var fileReader io.ReadCloser = in

		tr := tar.NewReader(fileReader)
		for {
			header, err := tr.Next()

			switch {
			case err == io.EOF:
				goto tarEof
			case err != nil:
				return err
			case header == nil:
				continue
			}

			// the target location where the dir/file should be created
			target := filepath.Join(dest, header.Name)

			// Check the file type
			switch header.Typeflag {

			// if its a dir and it doesn't exist create it
			case tar.TypeDir:
				if _, err := os.Stat(target); err != nil {
					if err := os.MkdirAll(target, 0755); err != nil {
						return err
					}
				}

				// handle creation of file
			case tar.TypeReg:
				f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}

				// copy over contents
				if _, err := io.Copy(f, tr); err != nil {
					return err
				}

				// manually close here after each file operation; defering would cause each
				// file close to wait until all operations have completed.
				f.Close()

			case tar.TypeSymlink:
				source := header.Linkname
				err := os.Symlink(source, target)
				if err != nil {
					return err
				}
			}
		}
	tarEof:
	}

	return ans
}
