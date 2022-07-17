package socket

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
)

var ErrUnsupportedBase = errors.New("unsuported base")
var ErrInvalidNumber = errors.New("value doesn't correspond with base")
var ErrInvalidPermissions = errors.New("invalid file permissions")

// Validate number corresponds with base
func validateBase(s string, base uint32) error {
	// Only bases up to 10 are supported, this might chante in the future.
	if base == 0 || base > 10 {
		return ErrUnsupportedBase
	}

	pattern := fmt.Sprintf("^[0-%v]{3}$", base-1)

	if match, err := regexp.MatchString(pattern, s); err != nil {
		return err // Invalid pattern
	} else if !match {
		return ErrInvalidNumber
	}

	return nil
}

type FileMode fs.FileMode

// MarshalJSON implements json.Marshaler.
func (fm FileMode) MarshalJSON() ([]byte, error) {

	if fm > 511 {
		// Detect wrong permissions
		return nil, ErrInvalidPermissions
	}

	value := strconv.FormatUint(uint64(fm), 8)
	b, err := json.Marshal(value)

	l := len(b)
	if err != nil {
		return nil, err
	} else if l == 5 {
		// Prevent slice overflows in the following section
		return b, nil
	}

	// Add leading zeros to the string
	data := []byte(`"000"`)
	for i, x := range b[1 : len(b)-1] {
		j := 6 - l + i
		data[j] = x
	}

	return []byte(data), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (fm *FileMode) UnmarshalJSON(data []byte) (err error) {
	var value string
	var intVal uint64

	if err = json.Unmarshal(data, &value); err != nil {
		// probably it's an valid json
	} else if err = validateBase(value, 8); err != nil {
		err = ErrInvalidPermissions // Validate permissions format
	} else if intVal, err = strconv.ParseUint(value, 8, 32); err != nil {
		err = ErrInvalidPermissions // This should never fail
	} else {
		*fm = FileMode(uint32(intVal))
	}

	return
}
