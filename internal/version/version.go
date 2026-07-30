package version

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultDisplay = "1.00.0101.0000"

// Build values are normally injected with -ldflags.
var (
	Display   = defaultDisplay
	Commit    = "dev"
	BuildDate = "unknown"
)

type Version struct {
	Major  int `json:"major"`
	Year   int `json:"year"`
	Month  int `json:"month"`
	Day    int `json:"day"`
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

func Parse(s string) (Version, error) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "v"))
	if strings.Count(s, ".") == 2 {
		return parseTagBody(s)
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return Version{}, fmt.Errorf("invalid version %q: want 1.YY.MMDD.HHmm or v1.YY.MDDHHmm", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, err
	}
	yy, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, err
	}
	if len(parts[2]) != 4 || len(parts[3]) != 4 {
		return Version{}, fmt.Errorf("invalid display version %q", s)
	}
	mmdd, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, err
	}
	hhmm, err := strconv.Atoi(parts[3])
	if err != nil {
		return Version{}, err
	}
	return validate(Version{Major: major, Year: yy, Month: mmdd / 100, Day: mmdd % 100, Hour: hhmm / 100, Minute: hhmm % 100})
}

func parseTagBody(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid tag version %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, err
	}
	yy, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, err
	}
	patchN, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, err
	}
	compact := fmt.Sprintf("%08d", patchN)
	if len(compact) != 8 {
		return Version{}, fmt.Errorf("invalid compact date %q", parts[2])
	}
	mm, _ := strconv.Atoi(compact[0:2])
	dd, _ := strconv.Atoi(compact[2:4])
	hh, _ := strconv.Atoi(compact[4:6])
	min, _ := strconv.Atoi(compact[6:8])
	return validate(Version{Major: major, Year: yy, Month: mm, Day: dd, Hour: hh, Minute: min})
}

func validate(v Version) (Version, error) {
	if v.Major < 0 || v.Year < 0 || v.Year > 99 {
		return Version{}, fmt.Errorf("version components out of range")
	}
	t := time.Date(2000+v.Year, time.Month(v.Month), v.Day, v.Hour, v.Minute, 0, 0, time.UTC)
	if int(t.Month()) != v.Month || t.Day() != v.Day || t.Hour() != v.Hour || t.Minute() != v.Minute {
		return Version{}, fmt.Errorf("invalid date/time in version")
	}
	return v, nil
}

func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}
func (v Version) String() string {
	return fmt.Sprintf("%d.%02d.%02d%02d.%02d%02d", v.Major, v.Year, v.Month, v.Day, v.Hour, v.Minute)
}
func (v Version) Tag() string {
	compact := fmt.Sprintf("%02d%02d%02d%02d", v.Month, v.Day, v.Hour, v.Minute)
	n, _ := strconv.Atoi(compact)
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Year, n)
}
func (v Version) Compare(o Version) int {
	a := []int{v.Major, v.Year, v.Month, v.Day, v.Hour, v.Minute}
	b := []int{o.Major, o.Year, o.Month, o.Day, o.Hour, o.Minute}
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
func Compare(a, b string) (int, error) {
	av, err := Parse(a)
	if err != nil {
		return 0, err
	}
	bv, err := Parse(b)
	if err != nil {
		return 0, err
	}
	return av.Compare(bv), nil
}
func Current() Version {
	v, err := Parse(Display)
	if err != nil {
		return MustParse(defaultDisplay)
	}
	return v
}
func JSON() ([]byte, error) {
	return json.MarshalIndent(map[string]string{"version": Current().String(), "tag": Current().Tag(), "commit": Commit, "build_date": BuildDate}, "", "  ")
}
