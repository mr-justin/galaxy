package utils

import (
	"strconv"
	"strings"
)

// VersionedMap is a CRDT where each key contains a version history of prior values.
// The value of the key is the value with the latest version.  VersionMaps can be combined
// such that they always converge to the same values for all keys.
type VersionedMap struct {
	values map[string][]mapEntry
}

type mapEntry struct {
	value   string
	version int64
}

func NewVersionedMap() *VersionedMap {
	return &VersionedMap{
		values: make(map[string][]mapEntry),
	}
}

func (v *VersionedMap) Set(key, value string, version int64) {
	entries := v.values[key]
	v.values[key] = append(entries, mapEntry{
		value:   value,
		version: version,
	})
}

func (v *VersionedMap) UnSet(key string, version int64) {
	entries := v.values[key]
	v.values[key] = append(entries, mapEntry{
		value:   "",
		version: version,
	})
}

func (v *VersionedMap) Get(key string) string {
	entries := v.values[key]
	maxEntry := mapEntry{}
	for _, entry := range entries {
		// value is max(version)
		if entry.version > maxEntry.version {
			maxEntry = entry
		}

		// if there is a conflict, prefer setting a value over unsetting one
		// as well the largest value as a tie-breaker if two sets conflict.
		if entry.version == maxEntry.version && entry.value > maxEntry.value {
			maxEntry = entry
		}

	}
	return maxEntry.value
}

func (v *VersionedMap) Merge(other *VersionedMap) {
	for k, entries := range other.values {
		v.values[k] = append(v.values[k], entries...)
	}
}

func (v *VersionedMap) MarshalMap() map[string]string {
	result := make(map[string]string)
	for key, entries := range v.values {
		for _, mapEntry := range entries {
			op := "s"
			if mapEntry.value == "" {
				op = "u"
			}
			mapKey := strings.Join([]string{key, op, strconv.FormatInt(mapEntry.version, 10)}, ":")
			result[mapKey] = mapEntry.value
		}

	}
	return result
}

func (v *VersionedMap) UnmarshalMap(serialized map[string]string) error {

	for key, val := range serialized {
		parts := strings.Split(key, ":")
		version, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return err
		}
		if parts[1] == "s" {
			v.Set(parts[0], val, version)
		} else {
			v.UnSet(parts[0], version)
		}
	}
	return nil
}