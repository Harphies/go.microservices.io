package data_structures

import (
	"bytes"
	"encoding/json"
)

// CustomDataHolder Use map to hold dynamic data that its structure. i.e. is not pre-known. Ref - https://www.sohamkamani.com/golang/json/#decoding-json-to-maps---unstructured-data
func CustomDataHolder() {

	// more usage
	//var r map[string]interface{}
	//_ = json.NewDecoder(res.Body).Decode(&r)
}

// InterfaceToByteArray convert any struct into byte array, alternative to json.Marshal(data)
func InterfaceToByteArray(data interface{}) ([]byte, error) {
	var b bytes.Buffer

	if err := json.NewEncoder(&b).Encode(data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
