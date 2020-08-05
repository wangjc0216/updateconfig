package main

import (
	"fmt"
	"testing"
)

func TestGenerateCfgStr(t *testing.T) {
	str, group, err := GenerateCfgStr(txconfigPath)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(str)
	fmt.Println(group)
}
