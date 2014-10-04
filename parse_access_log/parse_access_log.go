package main

import (
    "os"
    "bufio"
    "fmt"
    "io"
    "math"
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
   SCALE = 0
   EFFECTIVE_DIGIT = 3
   // for Apache(%D)
   // SCALE = -6
   // EFFECTIVE_DIGIT = 6
)

var (
    topCount = 10
    urlNormalizes = []string{
      "^GET /memo/[0-9]+$",
      "^GET /stylesheets/",
      "^GET /images/",
    }
)

type Measure struct {
  Url string
  Count int
  Total float64
  Min float64
  Mean float64
  Median float64
  P90 float64
  P95 float64
  P99 float64
  Max float64
}

type By func(a, b *Measure) bool

func (by By) Sort(measures []*Measure) {
  ms := &measureSorter{
    measures: measures,
    by: by,
  }
  sort.Sort(ms)
}

type measureSorter struct {
  measures []*Measure
  by func(a, b *Measure) bool
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
  Name string
  Summary string
  Sort By
}

var (
    columns = []*Column{
      &Column{ Name: "Count", Summary: "Count", Sort: func(a, b *Measure) bool { return a.Count > b.Count } },
      &Column{ Name: "Total", Summary: "Total", Sort: func(a, b *Measure) bool { return a.Total > b.Total } },
      &Column{ Name: "Mean", Summary: "Mean", Sort: func(a, b *Measure) bool { return a.Mean > b.Mean } },
      &Column{ Name: "Min", Summary: "Minimum(0 Percentile)", Sort: func(a, b *Measure) bool { return a.Min > b.Min } },
      &Column{ Name: "Median", Summary: "Median(50 Percentile)", Sort: func(a, b *Measure) bool { return a.Median > b.Median } },
      &Column{ Name: "P90", Summary: "90 Percentile", Sort: func(a, b *Measure) bool { return a.P90 > b.P90 } },
      &Column{ Name: "P95", Summary: "95 Percentile", Sort: func(a, b *Measure) bool { return a.P95 > b.P95 } },
      &Column{ Name: "P99", Summary: "99 Percentile", Sort: func(a, b *Measure) bool { return a.P99 > b.P99 } },
      &Column{ Name: "Max", Summary: "Maximum(100 Percentile)", Sort: func(a, b *Measure) bool { return a.Max > b.Max } },
    }
)

type Time struct {
  Url string
  Time float64
}

func showMeasures(measures []*Measure) {
  countWidth := 5 // for title
  totalWidth := 2 + EFFECTIVE_DIGIT
  meanWidth := 2 + EFFECTIVE_DIGIT * 2
  maxWidth := 2 + EFFECTIVE_DIGIT

  for i := 0; i < topCount; i++ {
    if countWidth < int(math.Log10(float64(measures[i].Count)) + 1) {
      countWidth = int(math.Log10(float64(measures[i].Count)) + 1)
    }
    if totalWidth < int(math.Log10(measures[i].Total) + 1 + EFFECTIVE_DIGIT + 1) {
      totalWidth = int(math.Log10(measures[i].Total) + 1 + EFFECTIVE_DIGIT + 1)
    }
    if meanWidth < int(math.Log10(measures[i].Max) + 1 + EFFECTIVE_DIGIT * 2 + 1) {
      meanWidth = int(math.Log10(measures[i].Max) + 1 + EFFECTIVE_DIGIT * 2 + 1)
    }
    if maxWidth < int(math.Log10(measures[i].Max) + 1 + EFFECTIVE_DIGIT + 1) {
      maxWidth = int(math.Log10(measures[i].Max) + 1 + EFFECTIVE_DIGIT + 1)
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
      format += fmt.Sprintf("%%%d.%df  ", meanWidth, EFFECTIVE_DIGIT * 2)
    default:
      fmt.Printf(fmt.Sprintf("%%%ds  ", maxWidth), column.Name)
      format += fmt.Sprintf("%%%d.%df  ", maxWidth, EFFECTIVE_DIGIT)
    }
  }
  fmt.Printf("url\n")
  format += "%s\n"

  for i := 0; i < topCount; i++ {
    m := measures[i]
    fmt.Printf(format, m.Count, m.Total, m.Mean, m.Min, m.Median, m.P90, m.P95, m.P99, m.Max, m.Url)
  }
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    if (useProfile) {
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
    times := make(map[string][]float64)
    go func() {
      for time := range ch {
        totals[time.Url] += time.Time
        times[time.Url] = append(times[time.Url], time.Time)
      }
    }()

    var wg sync.WaitGroup
    for {
        line, err := reader.ReadString('\n')
        if err == io.EOF {
          wg.Wait()
          close(ch)
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
            ch <- &Time{Url: url, Time: time}
          }
        }(line)
    }

    var measures []*Measure
    for url, total := range totals {
      sorted := times[url]
      sort.Float64s(sorted)
      count := len(sorted)
      measure := &Measure{
        Url: url,
        Count: count,
        Total: total,
        Min: sorted[0],
        Mean: totals[url]/float64(count),
        Median: sorted[int(count*50/100)],
        P90: sorted[int(count*90/100)],
        P95: sorted[int(count*95/100)],
        P99: sorted[int(count*99/100)],
        Max: sorted[count-1],
      }
      measures = append(measures, measure)
    }
    if len(measures) < topCount {
      topCount = len(measures)
    }

    for _, column := range columns {
      fmt.Printf("Sort By %s\n", column.Summary)
      By(column.Sort).Sort(measures)
      showMeasures(measures)
      fmt.Println()
    }
}
