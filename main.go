package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	useProfile = false
	// for Nginx($request_time)
	SCALE           = 0
	EFFECTIVE_DIGIT = 3
	// for Apache(%D)
	// SCALE = -6
	// EFFECTIVE_DIGIT = 6

	MIN_COUNT_WIDTH  = 5 // for title
	MIN_TOTAL_WIDTH  = 2 + EFFECTIVE_DIGIT
	MIN_MEAN_WIDTH   = 2 + EFFECTIVE_DIGIT*2
	MIN_MAX_WIDTH    = 2 + EFFECTIVE_DIGIT
	MIN_STATUS_WIDTH = 3 // for title
)

var (
	topCount      = 10
	allCount      = 37
	urlNormalizes = []string{
		"^GET /memo/[0-9]+$",
		"^GET /stylesheets/",
		"^GET /images/",
	}
)

type Measure struct {
	Url    string
	Count  int
	Total  float64
	Mean   float64
	Stddev float64
	Min    float64
	P50    float64
	P90    float64
	P95    float64
	P99    float64
	Max    float64
	S2xx   int
	S3xx   int
	S4xx   int
	S5xx   int
}

type By func(a, b *Measure) bool

func (by By) Sort(measures []*Measure) {
	ms := &measureSorter{
		measures: measures,
		by:       by,
	}
	sort.Sort(ms)
}

type measureSorter struct {
	measures []*Measure
	by       func(a, b *Measure) bool
}

func (s *measureSorter) Len() int {
	return len(s.measures)
}

func (s *measureSorter) Swap(i, j int) {
	s.measures[i], s.measures[j] = s.measures[j], s.measures[i]
}

func (s *measureSorter) Less(i, j int) bool {
	return s.by(s.measures[i], s.measures[j])
}

type Column struct {
	Name    string
	Summary string
	Sort    By
}

var (
	columns = []*Column{
		&Column{Name: "Count", Summary: "Count", Sort: func(a, b *Measure) bool { return a.Count > b.Count }},
		&Column{Name: "Total", Summary: "Total", Sort: func(a, b *Measure) bool { return a.Total > b.Total }},
		&Column{Name: "Mean", Summary: "Mean", Sort: func(a, b *Measure) bool { return a.Mean > b.Mean }},
		&Column{Name: "Stddev", Summary: "Standard Deviation", Sort: func(a, b *Measure) bool { return a.Stddev > b.Stddev }},
		&Column{Name: "Min"},
		&Column{Name: "P50"},
		&Column{Name: "P90"},
		&Column{Name: "P95"},
		&Column{Name: "P99"},
		&Column{Name: "Max", Summary: "Maximum(100 Percentile)", Sort: func(a, b *Measure) bool { return a.Max > b.Max }},
		&Column{Name: "2xx"},
		&Column{Name: "3xx"},
		&Column{Name: "4xx"},
		&Column{Name: "5xx"},
	}
)

type ByTime []*Time

type Time struct {
	Url        string
	Time       float64
	StatusCode int
}

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time > a[j].Time }

func getIntegerDigitWidth(f float64) int {
	var w int
	switch {
	case f < 0:
		w++
		fallthrough
	case math.Abs(f) < 1:
		w++
	default:
		w += int(math.Log10(math.Abs(f)) + 1)
	}
	return w
}

func showMeasures(measures []*Measure) {
	countWidth := MIN_COUNT_WIDTH
	totalWidth := MIN_TOTAL_WIDTH
	meanWidth := MIN_MEAN_WIDTH
	maxWidth := MIN_MAX_WIDTH
	s2xxWidth := MIN_STATUS_WIDTH
	s3xxWidth := MIN_STATUS_WIDTH
	s4xxWidth := MIN_STATUS_WIDTH
	s5xxWidth := MIN_STATUS_WIDTH

	for i := 0; i < topCount; i++ {
		var w int
		w = getIntegerDigitWidth(float64(measures[i].Count))
		if countWidth < w {
			countWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Total) + 1 + EFFECTIVE_DIGIT
		if totalWidth < w {
			totalWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Mean) + 1 + EFFECTIVE_DIGIT*2
		if meanWidth < w {
			meanWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Max) + 1 + EFFECTIVE_DIGIT
		if maxWidth < w {
			maxWidth = w
		}
		w = getIntegerDigitWidth(float64(measures[i].S2xx))
		if s2xxWidth < w {
			s2xxWidth = w
		}
		w = getIntegerDigitWidth(float64(measures[i].S3xx))
		if s3xxWidth < w {
			s3xxWidth = w
		}
		w = getIntegerDigitWidth(float64(measures[i].S4xx))
		if s4xxWidth < w {
			s4xxWidth = w
		}
		w = getIntegerDigitWidth(float64(measures[i].S5xx))
		if s5xxWidth < w {
			s5xxWidth = w
		}
	}

	var format string
	for _, column := range columns {
		switch column.Name {
		case "Count":
			fmt.Printf(fmt.Sprintf("%%%ds  ", countWidth), column.Name)
			format += fmt.Sprintf("%%%dd  ", countWidth)
		case "Total":
			fmt.Printf(fmt.Sprintf("%%%ds  ", totalWidth), column.Name)
			format += fmt.Sprintf("%%%d.%df  ", totalWidth, EFFECTIVE_DIGIT)
		case "Mean":
			fmt.Printf(fmt.Sprintf("%%%ds  ", meanWidth), column.Name)
			format += fmt.Sprintf("%%%d.%df  ", meanWidth, EFFECTIVE_DIGIT*2)
		case "Stddev":
			fmt.Printf(fmt.Sprintf("%%%ds  ", meanWidth), column.Name)
			format += fmt.Sprintf("%%%d.%df  ", meanWidth, EFFECTIVE_DIGIT*2)
		case "2xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s2xxWidth), column.Name)
			format += fmt.Sprintf("%%%dd  ", s2xxWidth)
		case "3xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s3xxWidth), column.Name)
			format += fmt.Sprintf("%%%dd  ", s3xxWidth)
		case "4xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s4xxWidth), column.Name)
			format += fmt.Sprintf("%%%dd  ", s4xxWidth)
		case "5xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s5xxWidth), column.Name)
			format += fmt.Sprintf("%%%dd  ", s5xxWidth)
		default:
			fmt.Printf(fmt.Sprintf("%%%ds  ", maxWidth), column.Name)
			format += fmt.Sprintf("%%%d.%df  ", maxWidth, EFFECTIVE_DIGIT)
		}
	}
	fmt.Printf("Url/Regexp\n")
	format += "%s\n"

	for i := 0; i < topCount; i++ {
		m := measures[i]
		fmt.Printf(format, m.Count, m.Total, m.Mean, m.Stddev, m.Min, m.P50, m.P90, m.P95, m.P99, m.Max, m.S2xx, m.S3xx, m.S4xx, m.S5xx, m.Url)
	}
}

func showTop(allTimes []*Time) {
	sort.Sort(ByTime(allTimes))
	if len(allTimes) < allCount {
		allCount = len(allTimes)
	}

	iWidth := getIntegerDigitWidth(float64(allCount))
	topWidth := getIntegerDigitWidth(allTimes[0].Time) + 1 + EFFECTIVE_DIGIT
	f := fmt.Sprintf("%%%dd  %%%d.%df  %%s\n", iWidth, topWidth, EFFECTIVE_DIGIT)
	for i := 0; i < allCount; i++ {
		fmt.Printf(f, i+1, allTimes[i].Time, allTimes[i].Url)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if useProfile {
		f, err := os.Create("/tmp/parse_access_log.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	reader := bufio.NewReaderSize(os.Stdin, 4096)
	scale := math.Pow10(SCALE)

	var urlNormalizeRegexps []*regexp.Regexp
	for _, str := range urlNormalizes {
		re := regexp.MustCompile(str)
		urlNormalizeRegexps = append(urlNormalizeRegexps, re)
	}

	ch := make(chan *Time)
	totals := make(map[string]float64)
	stddevs := make(map[string]float64)
	times := make(map[string][]float64)
	statusCode := make(map[string][]int)
	var allTimes []*Time

	var stddevWg sync.WaitGroup
	stddevWg.Add(1)
	go func() {
		defer stddevWg.Done()
		for time := range ch {
			totals[time.Url] += time.Time
			times[time.Url] = append(times[time.Url], time.Time)
			allTimes = append(allTimes, time)
			if statusCode[time.Url] == nil {
				statusCode[time.Url] = make([]int, 6)
			}
			statusCode[time.Url][time.StatusCode]++
		}
		for url, total := range totals {
			mean := total / float64(len(times[url]))
			for _, t := range times[url] {
				stddevs[url] += math.Pow(t-mean, 2)
			}
		}
	}()

	var wg sync.WaitGroup
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		wg.Add(1)
		go func(line string) {
			defer wg.Done()
			s := strings.Split(line, " ")
			if len(s) >= 7 {
				url := strings.TrimLeft(strings.Join(s[5:7], " "), "\"")
				for _, re := range urlNormalizeRegexps {
					if re.MatchString(url) {
						url = re.String()
						break
					}
				}
				time, err := strconv.ParseFloat(strings.Trim(s[len(s)-1], "\r\n"), 10)
				if err == nil {
					time = time * scale
				} else {
					time = 0.000
				}
				statusCode, err := strconv.Atoi(string(s[8][0]))
				if err != nil {
					statusCode = 0
				}
				ch <- &Time{Url: url, Time: time, StatusCode: statusCode}
			}
		}(line)
	}
	wg.Wait()
	close(ch)
	stddevWg.Wait()

	var measures []*Measure
	for url, total := range totals {
		sorted := times[url]
		sort.Float64s(sorted)
		count := len(sorted)
		measure := &Measure{
			Url:    url,
			Count:  count,
			Total:  total,
			Mean:   totals[url] / float64(count),
			Stddev: math.Sqrt(stddevs[url] / float64(count)),
			Min:    sorted[0],
			P50:    sorted[int(count*50/100)],
			P90:    sorted[int(count*90/100)],
			P95:    sorted[int(count*95/100)],
			P99:    sorted[int(count*99/100)],
			Max:    sorted[count-1],
			S2xx:   statusCode[url][2],
			S3xx:   statusCode[url][3],
			S4xx:   statusCode[url][4],
			S5xx:   statusCode[url][5],
		}
		measures = append(measures, measure)
	}
	if len(measures) < topCount {
		topCount = len(measures)
	}

	for _, column := range columns {
		if column.Sort != nil {
			fmt.Printf("Sort By %s\n", column.Summary)
			By(column.Sort).Sort(measures)
			showMeasures(measures)
			fmt.Println()
		}
	}

	fmt.Printf("TOP %d Slow Requests\n", allCount)
	showTop(allTimes)
}
