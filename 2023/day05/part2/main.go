package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed sample.txt
var sample string

//go:embed input.txt
var input string

type Almanac struct {
	SeedRanges []SeedRange
	Maps       []AlmanacMap
	Log        io.Writer
}

func (a Almanac) TotalSeeds() (tot int) {
	for _, r := range a.SeedRanges {
		tot = tot + r.Range
	}
	return tot
}

func (a Almanac) LowestLocationThreaded(n int) int {
	start := time.Now()
	inputCh := make(chan int)
	outputCh := make(chan int, len(a.SeedRanges))
	for i := 0; i < n; i++ {
		go func(workerIdx int) {
			for rangeIdx := range inputCh {
				start := time.Now()
				r := a.SeedRanges[rangeIdx]
				res := a.LowestLocationBySeedRangeIdx(rangeIdx)
				fmt.Printf("[worker: %d] seed range #%d (%d -> %d) = %d %s\n", workerIdx, rangeIdx, r.Start, r.End(), res, time.Since(start))
				outputCh <- res
			}
		}(i)
	}

	for idx := range a.SeedRanges {
		inputCh <- idx
	}

	lowest := -1
	for i := 0; i < len(a.SeedRanges); i++ {
		res := <-outputCh
		if lowest < 0 || res < lowest {
			lowest = res
		}
	}
	elapsed := time.Since(start)
	fmt.Println("RESULT:", lowest, elapsed)
	return lowest
}

func (a Almanac) LowestLocationThreadedBatched(n int, batchSize int) int {
	start := time.Now()
	inputCh := make(chan SeedRange)
	outputCh := make(chan int)

	// Create workers
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(workerIdx int) {
			// When inputCh is closed, there is no more work to do
			for r := range inputCh {
				res := a.LowestLocationByRange(r.Start, r.End())
				outputCh <- res
			}
			wg.Done()
		}(i)
	}

	// Start a thread to start feeding work to said workers
	go func() {
		for idx, r := range a.SeedRanges {
			start := time.Now()
			for b := r.Start; b < r.End(); b += batchSize {
				inputCh <- SeedRange{
					Start: b,
					Range: min(batchSize, r.Range),
				}
			}
			fmt.Printf("Range %d of %d (%d->%d) %s\n", idx+1, len(a.SeedRanges), r.Start, r.End(), time.Since(start))
		}
		// All work dispatched, close input channel
		close(inputCh)
		// Wait for all workers to cease
		wg.Wait()
		// Close output channel to retrieve result
		close(outputCh)
	}()

	// Read all results
	lowest := -1
	for res := range outputCh {
		if lowest < 0 || res < lowest {
			lowest = res
		}
	}
	elapsed := time.Since(start)
	fmt.Println("RESULT:", lowest, elapsed)
	return lowest
}

func (a Almanac) LowestLocation() int {
	lowest := -1
	for idx, r := range a.SeedRanges {
		fmt.Printf("seed range: %d -> %d = ", r.Start, r.End())
		start := time.Now()
		lowestInRange := a.LowestLocationBySeedRangeIdx(idx)
		elapsed := time.Since(start)
		fmt.Printf("%d Elapsed: %s Per Seed: ~%s\n", lowestInRange, elapsed, elapsed/time.Duration(r.Range))
		if lowest < 0 || lowestInRange < lowestInRange {
			lowest = lowestInRange
		}
	}
	return lowest
}

func (a Almanac) LowestLocationBySeedRangeIdx(idx int) int {
	r := a.SeedRanges[idx]
	return a.LowestLocationByRange(r.Start, r.End())
}

func (a Almanac) LowestLocationByRange(start int, end int) int {
	lowest := -1
	for seed := start; seed < end; seed++ {
		location := a.Lookup(seed)
		if lowest < 0 || location < lowest {
			lowest = location
		}
	}
	return lowest
}

func (a Almanac) Lookup(seed int) (location int) {
	location = seed
	fmt.Fprintf(a.Log, "seed: %d", seed)
	for _, m := range a.Maps {
		location = m.Lookup(location)
		fmt.Fprintf(a.Log, " %s: %d", m.Output, location)
	}
	fmt.Fprintln(a.Log)
	return
}

type SeedRange struct {
	Start int
	Range int
}

func (r SeedRange) End() int {
	return r.Start + r.Range
}

type AlmanacMap struct {
	Input   string
	Output  string
	Entries []Entry
}

type Entry struct {
	Destination int
	Source      int
	SourceRange int
}

func (a AlmanacMap) Lookup(seed int) int {
	for _, e := range a.Entries {
		if seed >= e.Source && seed < e.Source+e.SourceRange {
			return seed - e.Source + e.Destination
		}
	}
	return seed
}

func LoadAlmanac(input string) (almanac Almanac) {
	almanac.Log = io.Discard
	scanner := bufio.NewScanner(bytes.NewBufferString(input))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "seeds: ") {
			rawSeeds := strings.Split(strings.TrimPrefix(scanner.Text(), "seeds: "), " ")
			for i := 0; i < len(rawSeeds); i += 2 {
				seedStart, err := strconv.Atoi(rawSeeds[i])
				if err != nil {
					panic(err)
				}
				seedRange, err := strconv.Atoi(rawSeeds[i+1])
				if err != nil {
					panic(err)
				}
				almanac.SeedRanges = append(almanac.SeedRanges, SeedRange{
					Start: seedStart,
					Range: seedRange,
				})
			}
		} else if strings.HasSuffix(line, " map:") {
			splitName := strings.Split(strings.TrimSuffix(line, " map:"), "-")
			aMap := AlmanacMap{
				Input:  splitName[0],
				Output: splitName[2],
			}
			for scanner.Scan() {
				if len(scanner.Text()) > 0 {
					var entry Entry
					if _, err := fmt.Fscanf(bytes.NewReader(scanner.Bytes()), "%d %d %d", &entry.Destination, &entry.Source, &entry.SourceRange); err != nil {
						panic(err)
					}
					aMap.Entries = append(aMap.Entries, entry)
				} else {
					break
				}
			}
			almanac.Maps = append(almanac.Maps, aMap)
		}
	}
	if scanner.Err() != nil {
		panic(scanner.Err())
	}
	return
}

func take1() {
	almanac := LoadAlmanac(input)
	fmt.Println("Seed ranges:", len(almanac.SeedRanges))
	fmt.Println("Total seeds:", almanac.TotalSeeds())

	start := time.Now()
	almanac.Lookup(2276375722)
	elapsed := time.Since(start)
	fmt.Println("Est time per lookup   :", elapsed)
	fmt.Println("Est time for soulution:", elapsed*time.Duration(almanac.TotalSeeds()))

	fmt.Println("Result  : ", almanac.LowestLocation())
}

func take2() {
	almanac := LoadAlmanac(input)
	almanac.LowestLocationThreaded(10)
	/**
	[worker: 8] seed range #7 (4207087048 -> 4243281404) = 859731507 50.191339125s
	[worker: 5] seed range #5 (1365412380 -> 1445496560) = 2444964298 1m49.935431s
	[worker: 4] seed range #1 (3424292843 -> 3506403140) = 46294175 1m50.524392458s
	[worker: 3] seed range #3 (3289792522 -> 3393308609) = 250131305 2m15.807620333s
	[worker: 9] seed range #0 (2276375722 -> 2436523854) = 593974860 3m10.913914375s
	[worker: 1] seed range #8 (1515742281 -> 1689752261) = 421684844 3m20.306702334s
	[worker: 0] seed range #9 (6434225 -> 298276999) = 228039921 4m48.190282125s
	[worker: 2] seed range #2 (1692203766 -> 2035017733) = 322357427 5m28.605480333s
	[worker: 7] seed range #6 (3574751516 -> 4159532652) = 146637026 8m14.946491917s
	[worker: 6] seed range #4 (2590548294 -> 3180906055) = 642813586 8m30.402436084s
	RESULT: 46294175 8m30.404119375s
	*/
}

func take3() {
	almanac := LoadAlmanac(input)
	almanac.LowestLocationThreadedBatched(10, 1_000_000)
	/**
	Range 1 of 10 (2276375722->2436523854) 23.187851875s
	Range 2 of 10 (3424292843->3506403140) 11.62737675s
	Range 3 of 10 (1692203766->2035017733) 49.7464365s
	Range 4 of 10 (3289792522->3393308609) 15.956941125s
	Range 5 of 10 (2590548294->3180906055) 1m25.378993083s
	Range 6 of 10 (1365412380->1445496560) 11.504526125s
	Range 7 of 10 (3574751516->4159532652) 1m23.6093285s
	Range 8 of 10 (4207087048->4243281404) 5.15630825s
	Range 9 of 10 (1515742281->1689752261) 25.336668875s
	Range 10 of 10 (6434225->298276999) 41.748108208s
	RESULT: 46294175 5m54.12367525s
	*/
}

func main() {
	//take1()
	//take2()
	take3()
}
