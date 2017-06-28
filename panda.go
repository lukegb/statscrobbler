// Copyright 2017 Luke Granger-Brown. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type PandaSource struct {
	id uint
}

type pandaResponse struct {
	ErrNo  int    `json:"errno"`
	ErrMsg string `json:"errmsg"`
	Data   struct {
		RoomInfo struct {
			PersonNum string `json:"person_num"`
		} `json:"roominfo"`
	} `json:"data"`
}

func (s *PandaSource) GetViewCount() (uint64, error) {
	resp, err := http.Get(fmt.Sprintf("http://www.panda.tv/api_room?roomid=%d", s.id))
	if err != nil {
		return 0, fmt.Errorf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	var x pandaResponse
	if err := json.NewDecoder(resp.Body).Decode(&x); err != nil {
		return 0, fmt.Errorf("json.Decode: %v", err)
	}

	num, err := strconv.ParseUint(x.Data.RoomInfo.PersonNum, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("ParseUint(%q): %v", x.Data.RoomInfo.PersonNum, err)
	}

	return num, nil
}

func NewPandaSource(videoID uint) (ViewCountSource, error) {
	return &PandaSource{
		videoID,
	}, nil
}
