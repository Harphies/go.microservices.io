package data_structures

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"strconv"
)

func BufferToString() string {
	var b bytes.Buffer
	b.WriteString("G")
	return b.String()
}

func ByArrayToString(data []byte) string {
	return string(data)
}

// https://yourbasic.org/golang/io-reader-interface-explained/
// https://golang.cafe/blog/golang-convert-byte-slice-to-io-reader.html

func StructToIoReader(item interface{}) io.Reader {
	// First convert to byte array
	data, _ := json.Marshal(item)

	// then byte array to io reader type
	reader := bytes.NewReader(data)

	return reader

}

// StructToString Converts struct to string
func StructToString(item string) string {
	// First convert to byte array
	data, _ := json.Marshal(item)

	// then convert byte array to string
	return string(data)
}

// MapToString converts an arbitrary go object to string
func MapToString(m interface{}) string {
	var b bytes.Buffer

	// Note: gob encoding is faster than json:
	if err := json.NewEncoder(&b).Encode(m); err != nil {
		return "Error In encoding"
	}
	return string(b.Bytes())
}

// MapToByteArray converts an arbitrary go object to byte Array
func MapToByteArray(m interface{}) []byte {
	var b bytes.Buffer

	// Note: gob encoding is faster than json:
	if err := gob.NewEncoder(&b).Encode(m); err != nil {
		return []byte("Error In encoding")
	}
	return b.Bytes()
}

func StringToInt(yourString string) (int64, error) {
	data, err := strconv.ParseInt(yourString, 10, 64)
	if err != nil {
		return 0, err
	}
	return data, nil
}
