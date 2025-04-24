package main

import (
	"fmt"
	"testing"
)

func TestComputeHmacSha1(t *testing.T) {
	secret := "HLkPcWmycL57mfJt"
	message := "POST&/api/login&1743389682742"
	expected := "5+RxBM+QQ7+DCj0PRU0iC5M+OaI="
	result := ComputeHmacSha1(message, secret)
	if result != expected {
		t.Errorf("Expected %s but got %s", expected, result)
	} else {
		fmt.Println("Test passed!")
	}

}
