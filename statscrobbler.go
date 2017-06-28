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
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	youtubeAPIKey = flag.String("youtube-api-key", "", "YouTube API Key")
)

const (
	historicalFilename = "scrobbler.historical.json"
	configFilename     = "scrobbler.config.json"
)

type ViewCountSource interface {
	GetViewCount() (uint64, error)
}

type dataPoint struct {
	Time       time.Time         `json:"timestamp"`
	ViewCounts map[string]uint64 `json:"viewCounts"`
}

func saveHistoricalData(vcs []dataPoint) error {
	f, err := os.Create(historicalFilename)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	if err := enc.Encode(vcs); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

func loadHistoricalData() ([]dataPoint, error) {
	f, err := os.Open(historicalFilename)
	if err != nil {
		log.Printf("failed to open %s: %v", historicalFilename, err)
		return nil, nil
	}

	var dps []dataPoint
	if err := json.NewDecoder(f).Decode(&dps); err != nil {
		return nil, err
	}
	return dps, nil
}

type Config struct {
	YouTube map[string]string
	Panda   map[string]uint
}

func loadConfig() (map[string]ViewCountSource, error) {
	f, err := os.Open(configFilename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	srcs := make(map[string]ViewCountSource)
	for n, vidID := range cfg.YouTube {
		src, err := NewYouTubeSource(*youtubeAPIKey, vidID)
		if err != nil {
			return nil, fmt.Errorf("%s: NewYouTubeSource(%q): %v", n, vidID, err)
		}

		srcs[n] = src
	}

	for n, vidID := range cfg.Panda {
		src, err := NewPandaSource(vidID)
		if err != nil {
			return nil, fmt.Errorf("%s: NewPandaSource(%d): %v", n, vidID, err)
		}

		srcs[n] = src
	}

	return srcs, nil
}

func main() {
	flag.Parse()

	srcs, err := loadConfig()
	if err != nil {
		log.Fatalf("loadConfig: %v", err)
	}

	viewCounts, err := loadHistoricalData()
	if err != nil {
		log.Fatalf("loadHistoricalData: %v", err)
	}

	updateViewCount := func() {
		vcs := make(map[string]uint64)
		for n, src := range srcs {
			vc, err := src.GetViewCount()
			if err != nil {
				log.Printf("%s: %v", err)
				continue
			}
			vcs[n] = vc
		}

		viewCounts = append(viewCounts, dataPoint{time.Now(), vcs})
		if err := saveHistoricalData(viewCounts); err != nil {
			log.Printf("failed to saveHistoricalData: %v", err)
		}
		log.Println(vcs)
	}

	if viewCounts == nil {
		updateViewCount()
	}
	go func() {
		t := time.NewTicker(20 * time.Second)
		for range t.C {
			updateViewCount()
		}
	}()

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(viewCounts)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "%s", `<!DOCTYPE html>
<html>
<head>
<style>
html, body {
	width: 100vw;
	height: 100vh;
}
.chart {
	position: absolute;
	top: 2%; left: 2%;
	bottom: 2%; right: 22%;
}
.table {
	position: absolute;
	top: 2%; left: 82%;
	bottom: 2%; right: 2%;
}
</style>
<script type="text/javascript" src="https://www.gstatic.com/charts/loader.js"></script>
</head>
<body>
<div class="chart" id="ch"></div>
<div class="table" id="tb"></div>
<script>
(function() {
	"use strict";


	google.charts.load('current', {packages: ['line']});
	google.charts.setOnLoadCallback(function() {
		var options = {
			animation: {
				duration: 1000,
				easing: 'out',
			},
			hAxis: {
				format: 'HH:mm',
			},
		};

		var cols = [];
		var data = new google.visualization.DataTable();
		var $tbl = document.querySelector('#tb');
		data.addColumn('datetime', 'Date');

		var chart = new google.charts.Line(document.querySelector('#ch'));
		var latestTS = null;

		var redraw = function() {
			fetch('/data').then(r => r.json()).then(js => {
				for (var n = 0; n < js.length; n++) {
					var j = js[n];
					var date = +new Date(j.timestamp);
					if (latestTS && date < latestTS) {
						continue;
					}
					latestTS = date;

					var d = [new Date(j.timestamp)];
					for (var k of Object.keys(j.viewCounts)) {
						var idx = cols.indexOf(k);
						if (idx == -1) {
							idx = cols.length;
							cols.push(k);
							data.addColumn('number', k);
						}

						d[idx+1] = j.viewCounts[k];
					}
					console.log(d);
					data.addRow(d);
				}
				chart.draw(data, google.charts.Line.convertOptions(options));

				var j = js[js.length-1];
				var tbl = document.createElement('table');

				var hds = ['Stream', 'Min', 'Max', 'Current'];
				var tr = document.createElement('tr');
				tbl.appendChild(tr);
				for (var hd of hds) {
					var th = document.createElement('th');
					th.textContent = hd;
					tr.appendChild(th);
				}

				var r = data.getNumberOfRows()-1;
				for (var i = 1; i < data.getNumberOfColumns(); i++) {
					var tr = document.createElement('tr');
					tbl.appendChild(tr);
					var th = document.createElement('th');
					th.textContent = data.getColumnLabel(i);
					tr.appendChild(th);

					var cr = data.getColumnRange(i);

					var td = document.createElement('td');
					td.textContent = cr.min;
					tr.appendChild(td);

					var td = document.createElement('td');
					td.textContent = cr.max;
					tr.appendChild(td);

					var td = document.createElement('td');
					td.textContent = data.getValue(r, i);
					tr.appendChild(td);
				}

				while ($tbl.firstChild) {
					$tbl.removeChild($tbl.firstChild);
				}
				$tbl.appendChild(tbl);

				setTimeout(redraw, 5000);
			});
		};

		redraw();
	});
})();
</script>
</body>
</html>
`)
	})
	http.ListenAndServe(":8989", nil)
}
