package socket

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strconv"
)

var ErrInvalidPermissions = errors.New("invalid file permissions")

type FileMode fs.FileMode

// MarshalJSON implements json.Marshaler.
func (fm FileMode) MarshalJSON() ([]byte, error) {

	if fm > 0777 {
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
	} else if len(value) != 3 {
		err = ErrInvalidPermissions // Validate permissions length
	} else if intVal, err = strconv.ParseUint(value, 8, 9); err != nil {
		err = ErrInvalidPermissions // Validate permissions format
	} else {
		*fm = FileMode(uint32(intVal))
	}

	return
}
