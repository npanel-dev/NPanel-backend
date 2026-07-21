package tool

import (
	"os"
	"testing"
)

func testRedisURI() string {
	if uri := os.Getenv("NPANEL_TEST_REDIS_URI"); uri != "" {
		return uri
	}
	return "redis://localhost:6379"
}

func TestParseRedisURI(t *testing.T) {
	uri := testRedisURI()
	addr, password, database, err := ParseRedisURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(addr, password, database)
}

func TestRedisPing(t *testing.T) {
	uri := testRedisURI()
	addr, password, database, err := ParseRedisURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	err = RedisPing(addr, password, database)
	if err != nil {
		t.Fatal(err)
	}
}
