package edp

import "testing"

func TestEncrypto(t *testing.T) {
	str := "007928_order"
	result, err := Encrypto(str)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}
