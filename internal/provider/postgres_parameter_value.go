package provider

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// unitUndefined is the API unit for numeric parameters that have no unit.
const unitUndefined = "UNDEFINED"

// parameterUnit maps an API unit enum to its postgresql.conf suffix and its
// multiplier to the family base unit: bytes for memory, microseconds for time.
type parameterUnit struct {
	apiName string
	suffix  string
	memory  bool
	factor  float64
}

// parameterUnits is ordered from smallest to largest within each family.
var parameterUnits = []parameterUnit{
	{apiName: "BYTES", suffix: "B", memory: true, factor: 1},
	{apiName: "KILOBYTES", suffix: "kB", memory: true, factor: 1 << 10},
	{apiName: "MEGABYTES", suffix: "MB", memory: true, factor: 1 << 20},
	{apiName: "GIGABYTES", suffix: "GB", memory: true, factor: 1 << 30},
	{apiName: "TERABYTES", suffix: "TB", memory: true, factor: 1 << 40},
	{apiName: "MICROSECONDS", suffix: "us", memory: false, factor: 1},
	{apiName: "MILLISECONDS", suffix: "ms", memory: false, factor: 1e3},
	{apiName: "SECONDS", suffix: "s", memory: false, factor: 1e6},
	{apiName: "MINUTES", suffix: "min", memory: false, factor: 60e6},
	{apiName: "HOURS", suffix: "h", memory: false, factor: 3600e6},
	{apiName: "DAYS", suffix: "d", memory: false, factor: 86400e6},
}

const validUnitSuffixes = "B, kB, MB, GB, TB, us, ms, s, min, h, d"

func unitByAPIName(name string) (parameterUnit, bool) {
	for _, u := range parameterUnits {
		if u.apiName == name {
			return u, true
		}
	}
	return parameterUnit{}, false
}

func unitBySuffix(suffix string) (parameterUnit, bool) {
	for _, u := range parameterUnits {
		if u.suffix == suffix {
			return u, true
		}
	}
	return parameterUnit{}, false
}

// nextSmallerUnit returns the previous unit of the same family, if any.
func nextSmallerUnit(u parameterUnit) (parameterUnit, bool) {
	for i, candidate := range parameterUnits {
		if candidate.apiName != u.apiName || i == 0 {
			continue
		}
		smaller := parameterUnits[i-1]
		if smaller.memory == u.memory {
			return smaller, true
		}
	}
	return parameterUnit{}, false
}

func familyName(u parameterUnit) string {
	if u.memory {
		return "memory"
	}
	return "time"
}

// parameterCatalogEntry is one parameter as reported by getPostgresParameters.
type parameterCatalogEntry struct {
	info    tsClient.ParameterInfo
	numeric bool
	// unit is the API unit enum for numeric parameters; unitUndefined for plain numbers.
	unit     string
	value    float64
	strValue string
}

// parsedParameterValue is a config value ready to be sent to setPostgresParameters.
type parsedParameterValue struct {
	numeric bool
	value   float64
	unit    string
	str     string
}

func buildParameterCatalog(p *tsClient.PostgresParameters) map[string]parameterCatalogEntry {
	catalog := make(map[string]parameterCatalogEntry, len(p.StringParameters)+len(p.NumericParameters))
	for _, sp := range p.StringParameters {
		catalog[sp.Info.Name] = parameterCatalogEntry{info: sp.Info, strValue: sp.CurrentValue}
	}
	for _, np := range p.NumericParameters {
		unit := np.Unit
		if unit == "" {
			unit = unitUndefined
		}
		catalog[np.Info.Name] = parameterCatalogEntry{info: np.Info, numeric: true, unit: unit, value: np.CurrentValue}
	}
	return catalog
}

var valueWithUnitRe = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*([A-Za-z]+)$`)

// parseParameterValue interprets a postgresql.conf style value for the given parameter.
func parseParameterValue(entry parameterCatalogEntry, raw string) (parsedParameterValue, error) {
	raw = strings.TrimSpace(raw)
	if !entry.numeric {
		return parsedParameterValue{str: raw}, nil
	}
	if entry.unit == unitUndefined {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return parsedParameterValue{}, fmt.Errorf("%q is not a number", raw)
		}
		return parsedParameterValue{numeric: true, value: v, unit: unitUndefined}, nil
	}

	entryUnit, ok := unitByAPIName(entry.unit)
	if !ok {
		return parsedParameterValue{}, fmt.Errorf("unsupported unit %q reported by the API", entry.unit)
	}

	// Bare numbers are ambiguous for unit parameters. Postgres reads 0 and negatives
	// as special values in the default unit, so only those are accepted bare.
	if v, err := strconv.ParseFloat(raw, 64); err == nil {
		if v > 0 {
			return parsedParameterValue{}, fmt.Errorf("%q needs an explicit unit, for example 64%s", raw, entryUnit.suffix)
		}
		return parsedParameterValue{numeric: true, value: v, unit: entry.unit}, nil
	}

	m := valueWithUnitRe.FindStringSubmatch(raw)
	if m == nil {
		return parsedParameterValue{}, fmt.Errorf("%q is not a number followed by a unit (valid units: %s)", raw, validUnitSuffixes)
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return parsedParameterValue{}, fmt.Errorf("%q is not a number", m[1])
	}
	u, ok := unitBySuffix(m[2])
	if !ok {
		return parsedParameterValue{}, fmt.Errorf("unknown unit %q (valid units: %s)", m[2], validUnitSuffixes)
	}
	if u.memory != entryUnit.memory {
		return parsedParameterValue{}, fmt.Errorf("unit %q is a %s unit but the parameter takes %s units", m[2], familyName(u), familyName(entryUnit))
	}

	// The API rejects fractional values with units, so express them in a smaller unit.
	for v != math.Trunc(v) {
		smaller, ok := nextSmallerUnit(u)
		if !ok {
			return parsedParameterValue{}, fmt.Errorf("%q cannot be expressed as a whole number of %s", raw, u.suffix)
		}
		v = v * u.factor / smaller.factor
		u = smaller
	}
	return parsedParameterValue{numeric: true, value: v, unit: u.apiName}, nil
}

// formatParameterValue renders the running value in postgresql.conf syntax using the API's unit.
func formatParameterValue(entry parameterCatalogEntry) string {
	if !entry.numeric {
		return entry.strValue
	}
	s := strconv.FormatFloat(entry.value, 'f', -1, 64)
	if entry.unit == unitUndefined {
		return s
	}
	u, ok := unitByAPIName(entry.unit)
	if !ok {
		return s
	}
	return s + u.suffix
}

// parameterValueEqual reports whether raw denotes the entry's running value.
func parameterValueEqual(entry parameterCatalogEntry, raw string) bool {
	p, err := parseParameterValue(entry, raw)
	if err != nil {
		return false
	}
	if !p.numeric {
		return p.str == entry.strValue
	}
	if p.unit == unitUndefined || entry.unit == unitUndefined {
		return p.unit == entry.unit && p.value == entry.value
	}
	pu, ok := unitByAPIName(p.unit)
	if !ok {
		return false
	}
	eu, ok := unitByAPIName(entry.unit)
	if !ok {
		return false
	}
	return p.value*pu.factor == entry.value*eu.factor
}
