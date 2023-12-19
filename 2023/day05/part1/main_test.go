package part1

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

//go:embed sample.txt
var sample string

//go:embed input.txt
var input string

type Almanac struct {
	Seeds []int
	Maps  []AlmanacMap
	Log   io.Writer
}

func (a Almanac) LowestLocation() int {
	lowest := -1
	for _, seed := range a.Seeds {
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
			for _, seedStr := range strings.Split(strings.TrimPrefix(scanner.Text(), "seeds: "), " ") {
				seed, err := strconv.Atoi(seedStr)
				if err != nil {
					panic(err)
				}
				almanac.Seeds = append(almanac.Seeds, seed)
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

func Test_Lookup(t *testing.T) {
	var almanac = LoadAlmanac(sample)
	almanac.Log = os.Stdout

	testCases := map[int]int{
		0:  0,
		1:  1,
		48: 48,
		49: 49,
		50: 52,
		51: 53,
		96: 98,
		97: 99,
		98: 50,
		99: 51,
	}
	for seed, soil := range testCases {
		if almanac.Maps[0].Lookup(seed) != soil {
			t.Fatalf("lookup of %d != %d", seed, soil)
		}
	}
}

func Test_sample(t *testing.T) {
	almanac := LoadAlmanac(sample)
	//almanac.Log = os.Stdout
	fmt.Println("Result: ", almanac.LowestLocation())
}

func Test_input(t *testing.T) {
	almanac := LoadAlmanac(input)
	//almanac.Log = os.Stdout
	fmt.Println("Result  : ", almanac.LowestLocation())
}
