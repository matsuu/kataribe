package main

import (
    "os"
    "bufio"
    "fmt"
    "io"
    "math"
    "regexp"
    "sort"
    "strconv"
    "strings"
)

const (
   EFFECTIVE_DIGIT = 3
)

type Measure struct {
  Url string
  Count int
  Total float64
  Min float64
  Mean float64
  Median float64
  P90 float64
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
    totals = make(map[string]float64)
    times = make(map[string][]float64)
    measures []*Measure
    topCount = 10
    columns = []*Column{
      &Column{ Name: "Count", Summary: "Count", Sort: func(a, b *Measure) bool { return a.Count > b.Count } },
      &Column{ Name: "Total", Summary: "Total", Sort: func(a, b *Measure) bool { return a.Total > b.Total } },
      &Column{ Name: "Mean", Summary: "Mean", Sort: func(a, b *Measure) bool { return a.Mean > b.Mean } },
      &Column{ Name: "Min", Summary: "Minimum(0 Percentile)", Sort: func(a, b *Measure) bool { return a.Min > b.Min } },
      &Column{ Name: "Median", Summary: "Median(50 Percentile)", Sort: func(a, b *Measure) bool { return a.Median > b.Median } },
      &Column{ Name: "P90", Summary: "90 Percentile", Sort: func(a, b *Measure) bool { return a.P90 > b.P90 } },
      &Column{ Name: "Max", Summary: "Maximum(100 Percentile)", Sort: func(a, b *Measure) bool { return a.Max > b.Max } },
    }
)

func showMeasures(measures []*Measure) {
  countWidth := 5 // for title
  totalWidth := 2 + EFFECTIVE_DIGIT
  maxWidth := 2 + EFFECTIVE_DIGIT

  for i := 0; i < topCount; i++ {
    if countWidth < int(math.Log10(float64(measures[i].Count)) + 1) {
      countWidth = int(math.Log10(float64(measures[i].Count)) + 1)
    }
    if totalWidth < int(math.Log10(measures[i].Total) + 1 + EFFECTIVE_DIGIT + 1) {
      totalWidth = int(math.Log10(measures[i].Total) + 1 + EFFECTIVE_DIGIT + 1)
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
    default:
      fmt.Printf(fmt.Sprintf("%%%ds  ", maxWidth), column.Name)
      format += fmt.Sprintf("%%%d.%df  ", maxWidth, EFFECTIVE_DIGIT)
    }
  }
  fmt.Printf("url\n")
  format += "%s\n"

  for i := 0; i < topCount; i++ {
    m := measures[i]
    fmt.Printf(format, m.Count, m.Total, m.Mean, m.Min, m.Median, m.P90, m.Max, m.Url)
  }
}

func main() {
    reader := bufio.NewReaderSize(os.Stdin, 4096)
    delimiter := regexp.MustCompile(" +")
    for {
        line, err := reader.ReadString('\n')
        if err == io.EOF {
          break
        } else if err != nil {
          panic(err)
        }
        s := delimiter.Split(line, -1)
        if len(s) > 0 {
          var url string
          if len(s) >= 7 {
            url = strings.TrimLeft(strings.Join(s[5:7], " "), "\"")
          }
          time, err := strconv.ParseFloat(strings.Trim(s[len(s)-1], "\r\n"), 10)
          if err != nil {
            time = 0.000
          }
          // time /= 1000000 // for Apache
          totals[url] += time
          times[url] = append(times[url], time)
        }
    }

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
