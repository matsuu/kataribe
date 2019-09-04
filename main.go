package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	RankingCount   int  `toml:"ranking_count"`
	SlowCount      int  `toml:"slow_count"`
	ShowStdDev     bool `toml:"show_stddev"`
	ShowStatusCode bool `toml:"show_status_code"`
	Percentiles    []float64
	Scale          int
	EffectiveDigit int    `toml:"effective_digit"`
	LogFormat      string `toml:"log_format"`
	RequestIndex   int    `toml:"request_index"`
	StatusIndex    int    `toml:"status_index"`
	DurationIndex  int    `toml:"duration_index"`
	Bundle         []bundleConfig
	Replace        []replaceConfig
	Bundles        map[string]bundleConfig // for backward compatibility

	ShowBytes  bool `toml:"show_bytes"`
	BytesIndex int  `toml:"bytes_index"`
}

type bundleConfig struct {
	Name   string
	Regexp string
}

type replaceConfig struct {
	Regexp  string
	Replace string
}

type Measure struct {
	Url         string
	Count       int
	Total       float64
	Mean        float64
	Stddev      float64
	Min         float64
	Percentiles []float64
	Max         float64
	S2xx        int
	S3xx        int
	S4xx        int
	S5xx        int

	TotalBytes int
	MinBytes   int
	MeanBytes  int
	MaxBytes   int
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
	columns []*Column
)

type ByTime []*Time

type Time struct {
	Url        string
	OriginUrl  string
	Time       float64
	StatusCode int
	Byte       int
}

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time > a[j].Time }

func buildColumns() {
	columns = append(columns, &Column{Name: "Count", Summary: "Count", Sort: func(a, b *Measure) bool { return a.Count > b.Count }})
	columns = append(columns, &Column{Name: "Total", Summary: "Total", Sort: func(a, b *Measure) bool { return a.Total > b.Total }})
	columns = append(columns, &Column{Name: "Mean", Summary: "Mean", Sort: func(a, b *Measure) bool { return a.Mean > b.Mean }})
	if config.ShowStdDev {
		columns = append(columns, &Column{Name: "Stddev", Summary: "Standard Deviation", Sort: func(a, b *Measure) bool { return a.Stddev > b.Stddev }})
	}
	columns = append(columns, &Column{Name: "Min"})
	for _, p := range config.Percentiles {
		name := fmt.Sprintf("P%2.1f", p)
		columns = append(columns, &Column{Name: name})
	}
	columns = append(columns, &Column{Name: "Max", Summary: "Maximum(100 Percentile)", Sort: func(a, b *Measure) bool { return a.Max > b.Max }})
	if config.ShowStatusCode {
		columns = append(columns, &Column{Name: "2xx"})
		columns = append(columns, &Column{Name: "3xx"})
		columns = append(columns, &Column{Name: "4xx"})
		columns = append(columns, &Column{Name: "5xx"})
	}
	if config.ShowBytes {
		columns = append(columns, &Column{Name: "TotalBytes"})
		columns = append(columns, &Column{Name: "MinBytes"})
		columns = append(columns, &Column{Name: "MeanBytes"})
		columns = append(columns, &Column{Name: "MaxBytes"})
	}
}

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
	MIN_COUNT_WIDTH := 5 // for title
	MIN_TOTAL_WIDTH := 2 + config.EffectiveDigit
	MIN_MEAN_WIDTH := 2 + config.EffectiveDigit*2
	MIN_MAX_WIDTH := 2 + config.EffectiveDigit
	MIN_STATUS_WIDTH := 3 // for title

	countWidth := MIN_COUNT_WIDTH // for title
	totalWidth := MIN_TOTAL_WIDTH
	meanWidth := MIN_MEAN_WIDTH
	maxWidth := MIN_MAX_WIDTH
	s2xxWidth := MIN_STATUS_WIDTH
	s3xxWidth := MIN_STATUS_WIDTH
	s4xxWidth := MIN_STATUS_WIDTH
	s5xxWidth := MIN_STATUS_WIDTH
	totalBytesWidth := 10
	bytesWidth := 9 // for title

	rankingCount := config.RankingCount
	if len(measures) < rankingCount {
		rankingCount = len(measures)
	}
	for i := 0; i < rankingCount; i++ {
		var w int
		w = getIntegerDigitWidth(float64(measures[i].Count))
		if countWidth < w {
			countWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Total) + 1 + config.EffectiveDigit
		if totalWidth < w {
			totalWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Mean) + 1 + config.EffectiveDigit*2
		if meanWidth < w {
			meanWidth = w
		}
		w = getIntegerDigitWidth(measures[i].Max) + 1 + config.EffectiveDigit
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
		w = getIntegerDigitWidth(float64(measures[i].TotalBytes))
		if totalBytesWidth < w {
			totalBytesWidth = w
		}
		w = getIntegerDigitWidth(float64(measures[i].MaxBytes))
		if bytesWidth < w {
			bytesWidth = w
		}
	}

	var formats []string
	for _, column := range columns {
		switch column.Name {
		case "Count":
			fmt.Printf(fmt.Sprintf("%%%ds  ", countWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", countWidth))
		case "Total":
			fmt.Printf(fmt.Sprintf("%%%ds  ", totalWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%d.%df  ", totalWidth, config.EffectiveDigit))
		case "Mean":
			fmt.Printf(fmt.Sprintf("%%%ds  ", meanWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%d.%df  ", meanWidth, config.EffectiveDigit*2))
		case "Stddev":
			fmt.Printf(fmt.Sprintf("%%%ds  ", meanWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%d.%df  ", meanWidth, config.EffectiveDigit*2))
		case "2xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s2xxWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", s2xxWidth))
		case "3xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s3xxWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", s3xxWidth))
		case "4xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s4xxWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", s4xxWidth))
		case "5xx":
			fmt.Printf(fmt.Sprintf("%%%ds  ", s5xxWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", s5xxWidth))
		case "TotalBytes":
			fmt.Printf(fmt.Sprintf("%%%ds  ", totalBytesWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", totalBytesWidth))
		case "MinBytes":
			fmt.Printf(fmt.Sprintf("%%%ds  ", bytesWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", bytesWidth))
		case "MeanBytes":
			fmt.Printf(fmt.Sprintf("%%%ds  ", bytesWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", bytesWidth))
		case "MaxBytes":
			fmt.Printf(fmt.Sprintf("%%%ds  ", bytesWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%dd  ", bytesWidth))

		default:
			fmt.Printf(fmt.Sprintf("%%%ds  ", maxWidth), column.Name)
			formats = append(formats, fmt.Sprintf("%%%d.%df  ", maxWidth, config.EffectiveDigit))
		}
	}
	fmt.Printf("Request\n")

	for r := 0; r < rankingCount; r++ {
		m := measures[r]
		c := 0
		fmt.Printf(formats[c], m.Count)
		c++
		fmt.Printf(formats[c], m.Total)
		c++
		fmt.Printf(formats[c], m.Mean)
		c++
		if config.ShowStdDev {
			fmt.Printf(formats[c], m.Stddev)
			c++
		}
		fmt.Printf(formats[c], m.Min)
		c++
		for i := range config.Percentiles {
			fmt.Printf(formats[c], m.Percentiles[i])
			c++
		}
		fmt.Printf(formats[c], m.Max)
		c++
		if config.ShowStatusCode {
			fmt.Printf(formats[c], m.S2xx)
			c++
			fmt.Printf(formats[c], m.S3xx)
			c++
			fmt.Printf(formats[c], m.S4xx)
			c++
			fmt.Printf(formats[c], m.S5xx)
			c++
		}
		if config.ShowBytes {
			fmt.Printf(formats[c], m.TotalBytes)
			c++
			fmt.Printf(formats[c], m.MinBytes)
			c++
			fmt.Printf(formats[c], m.MeanBytes)
			c++
			fmt.Printf(formats[c], m.MaxBytes)
			c++
		}

		fmt.Printf("%s\n", m.Url)
	}
}

func showTop(allTimes []*Time) {
	sort.Sort(ByTime(allTimes))
	slowCount := config.SlowCount
	if len(allTimes) < slowCount {
		slowCount = len(allTimes)
	}
	fmt.Printf("TOP %d Slow Requests\n", slowCount)

	iWidth := getIntegerDigitWidth(float64(slowCount))
	topWidth := getIntegerDigitWidth(allTimes[0].Time) + 1 + config.EffectiveDigit
	f := fmt.Sprintf("%%%dd  %%%d.%df  %%s\n", iWidth, topWidth, config.EffectiveDigit)
	for i := 0; i < slowCount; i++ {
		fmt.Printf(f, i+1, allTimes[i].Time, allTimes[i].OriginUrl)
	}
}

var configFile string
var config tomlConfig
var modeGenerate bool

func init() {
	const (
		defaultConfigFile = "kataribe.toml"
		usage             = "configuration file"
	)
	flag.StringVar(&configFile, "conf", defaultConfigFile, usage)
	flag.StringVar(&configFile, "f", defaultConfigFile, usage+" (shorthand)")
	flag.BoolVar(&modeGenerate, "generate", false, "generate "+usage)
	flag.Parse()
}

func main() {
	if modeGenerate {
		f, err := os.Create(configFile)
		if err != nil {
			log.Fatal("Failed to generate "+configFile+":", err)
		}
		defer f.Close()
		_, err = f.Write([]byte(CONFIG_TOML))
		if err != nil {
			log.Fatal("Failed to write "+configFile+":", err)
		}
		os.Exit(0)
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		fmt.Println(err)
		flag.Usage()
		return
	}

	reader := bufio.NewScanner(os.Stdin)
	scale := math.Pow10(config.Scale)

	done := make(chan struct{})

	urlNormalizeRegexps := make(map[string]*regexp.Regexp)

	chBundle := make(chan bundleConfig)
	go func() {
		for bundle := range chBundle {
			name := bundle.Name
			if name == "" {
				name = bundle.Regexp
			}
			urlNormalizeRegexps[name] = regexp.MustCompile(bundle.Regexp)
		}
		done <- struct{}{}
	}()

	for _, b := range config.Bundle {
		chBundle <- b
	}
	for _, b := range config.Bundles {
		chBundle <- b
	}

	type replaceRegexp struct {
		compiledRegexp *regexp.Regexp
		replace        string
	}
	urlReplaceRegexps := make([]*replaceRegexp, 0, len(config.Replace))
	chReplace := make(chan replaceConfig)
	go func() {
		for replace := range chReplace {
			urlReplaceRegexps = append(urlReplaceRegexps, &replaceRegexp{
				compiledRegexp: regexp.MustCompile(replace.Regexp),
				replace:        replace.Replace,
			})
		}
		done <- struct{}{}
	}()
	for _, r := range config.Replace {
		chReplace <- r
	}
	close(chBundle)
	<-done

	ch := make(chan *Time)
	totals := make(map[string]float64)
	stddevs := make(map[string]float64)
	times := make(map[string][]float64)
	totalBytes := make(map[string]int)
	bytes := make(map[string][]int)
	statusCode := make(map[string][]int)
	var allTimes []*Time

	go func() {
		for time := range ch {
			totals[time.Url] += time.Time
			times[time.Url] = append(times[time.Url], time.Time)
			allTimes = append(allTimes, time)
			totalBytes[time.Url] += time.Byte
			bytes[time.Url] = append(bytes[time.Url], time.Byte)
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
		done <- struct{}{}
	}()

	logParser := regexp.MustCompile(config.LogFormat)

	tasks := make(chan string)
	cpus := runtime.NumCPU()
	var wg sync.WaitGroup
	for worker := 0; worker < cpus; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for line := range tasks {
				submatch := logParser.FindAllStringSubmatch(strings.TrimSpace(line), -1)
				if len(submatch) > 0 {
					s := submatch[0]
					url := s[config.RequestIndex]
					originUrl := url
					for name, re := range urlNormalizeRegexps {
						if re.MatchString(url) {
							url = name
							break
						}
					}
					for _, replace := range urlReplaceRegexps {
						url = replace.compiledRegexp.ReplaceAllString(url, replace.replace)
					}
					time, err := strconv.ParseFloat(s[config.DurationIndex], 10)
					if err == nil {
						time = time * scale
					} else {
						time = 0.000
					}
					statusCode, err := strconv.Atoi(string(s[config.StatusIndex][0]))
					if err != nil {
						statusCode = 0
					}
					bytes, err := strconv.Atoi(s[config.BytesIndex])
					if err != nil {
						bytes = 0
					}
					ch <- &Time{Url: url, OriginUrl: originUrl, Time: time, StatusCode: statusCode, Byte: bytes}
				}
			}
		}()
	}

	for reader.Scan() {
		tasks <- reader.Text()
	}
	if err := reader.Err(); err != nil {
		log.Fatal("reading standard input:", err)
	}
	close(tasks)
	wg.Wait()
	close(ch)
	<-done

	var measures []*Measure
	for url, total := range totals {
		sorted := times[url]
		sort.Float64s(sorted)
		sortedBytes := bytes[url]
		sort.Ints(sortedBytes)
		count := len(sorted)
		var percentiles []float64
		for _, p := range config.Percentiles {
			percentiles = append(percentiles, sorted[int(float64(count)*p/100)])
		}

		measure := &Measure{
			Url:         url,
			Count:       count,
			Total:       total,
			Mean:        totals[url] / float64(count),
			Stddev:      math.Sqrt(stddevs[url] / float64(count)),
			Min:         sorted[0],
			Percentiles: percentiles,
			Max:         sorted[count-1],
			S2xx:        statusCode[url][2],
			S3xx:        statusCode[url][3],
			S4xx:        statusCode[url][4],
			S5xx:        statusCode[url][5],
			TotalBytes:  totalBytes[url],
			MinBytes:    sortedBytes[0],
			MeanBytes:   totalBytes[url] / count,
			MaxBytes:    sortedBytes[count-1],
		}
		measures = append(measures, measure)
	}

	if len(measures) > 0 {
		buildColumns()
		for _, column := range columns {
			if column.Sort != nil {
				fmt.Printf("Top %d Sort By %s\n", config.RankingCount, column.Summary)
				By(column.Sort).Sort(measures)
				showMeasures(measures)
				fmt.Println()
			}
		}
	}

	if len(allTimes) == 0 {
		log.Fatal("No parsed requests found. Please confirm log_format.")
	}
	showTop(allTimes)
}
