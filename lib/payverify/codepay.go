// Package payverify code payment signature tool (Easy Pay Standard)
//
// Signature algorithm:
//  1. Remove the sign and sign_type fields, and remove the null value fields
//  2. Sort by key in ascending dictionary order
//  3. Splice k1=v1&k2=v2…(remove the trailing &)
//  4. Append secret_key at the end
//  5. Get MD5 lowercase hex
package payverify

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// Sign generate signature
func Sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	first := true
	for _, k := range keys {
		v := params[k]
		if v == "" {
			continue
		}
		if !first {
			sb.WriteString("&")
		}
		first = false
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
	}
	sb.WriteString(secret)

	h := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(h[:])
}

// Verify Verify signature
func Verify(params map[string]string, secret string) bool {
	clientSign, ok := params["sign"]
	if !ok || clientSign == "" {
		return false
	}
	expected := Sign(params, secret)
	return clientSign == expected
}
