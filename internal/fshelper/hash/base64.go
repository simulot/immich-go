package hash

import "encoding/base64"

func Base64Encode(hash []byte, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}
