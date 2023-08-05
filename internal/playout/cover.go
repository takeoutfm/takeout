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

package playout

import (
	"net/http"
	"net/url"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/qeesung/image2ascii/ascii"
	"github.com/qeesung/image2ascii/convert"
	"github.com/takeoutfm/takeout/client"
	"github.com/takeoutfm/takeout/lib/header"
)

func asciiImage(context client.Context, url *url.URL, w, h int) ([][]ascii.CharPixel, error) {
	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = w
	convertOptions.FixedHeight = h

	converter := convert.NewImageConverter()
	img, err := openImage(context, url)
	if err != nil {
		return nil, err
	}
	return converter.Image2CharPixelMatrix(img, &convertOptions), nil
}

func openImage(context client.Context, url *url.URL) (image.Image, error) {
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(header.UserAgent, context.UserAgent())
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return img, nil
}
