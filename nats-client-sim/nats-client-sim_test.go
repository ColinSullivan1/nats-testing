package main

import "testing"

func Test_run(t *testing.T) {
	run("config.json", true, true, false, 0)
}
