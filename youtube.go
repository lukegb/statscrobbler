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
	"fmt"
	"log"
	"net/http"

	"google.golang.org/api/googleapi/transport"
	youtube "google.golang.org/api/youtube/v3"
)

type YouTubeSource struct {
	svc *youtube.Service
	id  string
}

func (s *YouTubeSource) GetViewCount() (uint64, error) {
	call := s.svc.Videos.List("liveStreamingDetails").Id(s.id)
	response, err := call.Do()
	if err != nil {
		return 0, err
	}

	if len(response.Items) != 1 {
		return 0, fmt.Errorf("youtube returned %d videos", len(response.Items))
	}

	vid := response.Items[0]
	return vid.LiveStreamingDetails.ConcurrentViewers, nil
}

func NewYouTubeSource(apiKey, videoID string) (ViewCountSource, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("youtube: API key not specified")
	}

	client := &http.Client{
		Transport: &transport.APIKey{Key: apiKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("youtube.New: %v", err)
	}

	return &YouTubeSource{
		svc: service,
		id:  videoID,
	}, nil
}
