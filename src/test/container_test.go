package test

import "io/ioutil"
import "os"
import "path/filepath"
import "testing"

func TestInContainer(t *testing.T) {
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Errorf("Couldn't make path absolute: %s", err)
	} else if abs != "/tmp/test/container_test" {
		t.Errorf("Looks like we're not running inside a container: %s", os.Args[0])
	}
}

func TestContainerData(t *testing.T) {
	data, err := ioutil.ReadFile("src/test/test_data/container_data.txt")
	if err != nil {
		t.Errorf("Failed to read data file: %s", err)
	} else {
		expected := "This file will only appear in the container if data is working properly.\n"
		if string(data) != expected {
			t.Errorf("Unexpected file contents: expected [%s], was [%s]", expected, string(data))
		}
	}
}
