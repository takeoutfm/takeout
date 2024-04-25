// Copyright 2023 defsub
//
// This file is part of TakeoutFM.
//
// TakeoutFM is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// TakeoutFM is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with TakeoutFM.  If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/takeoutfm/takeout/lib/header"
	"github.com/takeoutfm/takeout/lib/str"
)

type object struct {
	created     time.Time
	user        string
	contentType string
	data        []byte
}

type objectList struct {
	Objects []string
}

var (
	objects = make(map[string]map[string]object)
)

func apiObjectPut(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	user := ctx.User().Name
	if len(objects[user]) == 0 {
		objects[user] = make(map[string]object)
	}
	uuid := r.URL.Query().Get(ParamUUID)
	if uuid == "" {
		badRequest(w, ErrInvalidUUID)
		return
	}
	// only allow 1MB
	if r.ContentLength == -1 || r.ContentLength > 1*1024*1024 {
		badRequest(w, ErrInvalidContent)
		return
	}
	contentType := r.Header.Get(header.ContentType)
	if contentType == "" {
		badRequest(w, ErrInvalidContent)
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	if contentType == ApplicationJson {
		// validate the json
		result := make(map[string]interface{})
		err := json.Unmarshal(body, &result)
		if err != nil {
			badRequest(w, ErrInvalidContent)
			return
		}
	}
	// TODO validate yaml, others?
	objects[user][uuid] = object{
		created:     time.Now(),
		user:        user,
		contentType: contentType,
		data:        body,
	}
	w.WriteHeader(http.StatusCreated)
}

func apiObjectGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	user := ctx.User().Name
	uuid := r.URL.Query().Get(ParamUUID)
	if uuid == "" {
		badRequest(w, ErrInvalidUUID)
		return
	}
	if len(objects[user]) == 0 {
		notFoundErr(w)
		return
	}
	o, ok := objects[user][uuid]
	if ok {
		w.Header().Set(header.ContentType, o.contentType)
		w.Header().Set(header.ContentLength, str.Itoa(len(o.data)))
		w.Write(o.data)
	} else {
		notFoundErr(w)
	}
}

func apiObjectsList(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	user := ctx.User().Name
	if len(objects[user]) == 0 {
		notFoundErr(w)
		return
	}
	var list objectList
	for k := range objects[user] {
		list.Objects = append(list.Objects, k)
	}
	w.Header().Set(header.ContentType, ApplicationJson)
	json.NewEncoder(w).Encode(list)
}
