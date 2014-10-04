package main

import (
    "os"
    "bufio"
    "fmt"
    "io"
    "regexp"
    "sort"
    "strconv"
    "strings"
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

type Measures []*Measure

func (m Measures) Len() int { return len(m) }
func (m Measures) Swap(i, j int) { m[i], m[j] = m[j], m[i] }

type ByCount struct { Measures }
func (m ByCount) Less(i, j int) bool { return m.Measures[i].Count > m.Measures[j].Count }

type ByTotal struct { Measures }
func (m ByTotal) Less(i, j int) bool { return m.Measures[i].Total > m.Measures[j].Total }

type ByMin struct { Measures }
func (m ByMin) Less(i, j int) bool { return m.Measures[i].Min > m.Measures[j].Min }

type ByMean struct { Measures }
func (m ByMean) Less(i, j int) bool { return m.Measures[i].Mean > m.Measures[j].Mean }

type ByMedian struct { Measures }
func (m ByMedian) Less(i, j int) bool { return m.Measures[i].Median > m.Measures[j].Median }

type ByP90 struct { Measures }
func (m ByP90) Less(i, j int) bool { return m.Measures[i].P90 > m.Measures[j].P90 }

type ByMax struct { Measures }
func (m ByMax) Less(i, j int) bool { return m.Measures[i].Max > m.Measures[j].Max }

var (
    totals = make(map[string]float64)
    times = make(map[string][]float64)
    measures Measures
    topCount = 10
)

func showMeasures(measures []*Measure) {
    fmt.Printf("%8s %8s %8s %8s %8s %8s %8s %s\n", "count", "total", "min", "mean", "median", "p90", "max", "url")
    for i := 0; i < topCount; i++ {
      m := measures[i]
      fmt.Printf("%8d %8.3f %8.3f %8.3f %8.3f %8.3f %8.3f %s\n", m.Count, m.Total, m.Min, m.Mean, m.Median, m.P90, m.Max, m.Url)
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

    fmt.Println("Count")
    sort.Sort(ByCount{measures})
    showMeasures(measures)

    fmt.Println("Total")
    sort.Sort(ByTotal{measures})
    showMeasures(measures)

    fmt.Println("Min")
    sort.Sort(ByMin{measures})
    showMeasures(measures)

    fmt.Println("Mean")
    sort.Sort(ByMean{measures})
    showMeasures(measures)

    fmt.Println("Median(50Percentile)")
    sort.Sort(ByMedian{measures})
    showMeasures(measures)

    fmt.Println("90Percentile")
    sort.Sort(ByP90{measures})
    showMeasures(measures)

    fmt.Println("Max")
    sort.Sort(ByMax{measures})
    showMeasures(measures)
}
