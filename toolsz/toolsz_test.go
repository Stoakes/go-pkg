package toolsz_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Stoakes/go-pkg/log"
	"github.com/Stoakes/go-pkg/toolsz"
)

func TestStart(t *testing.T) {

	port := 51518
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	logger := log.Setup(ctx, log.Options{})

	toolServer := toolsz.New(port, logger, &map[string]http.Handler{
		"404": http.NotFoundHandler(),
	})
	go func() {
		err := toolServer.Start(ctx)
		if err != nil {
			t.Fatalf("Cannot start tools server: " + err.Error())
		}
	}()
	time.Sleep(100 * time.Millisecond)
	host := "http://localhost:" + strconv.Itoa(port)

	/* Test GET /metrics */
	body, err := testGet(host+"/metrics", http.StatusOK)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(string(body), "goroutines") {
		t.Errorf("Cannot find 'goroutines' in /metrics content")
	}

	/* Test GET /info */
	body, err = testGet(host+"/info", http.StatusOK)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(string(body), "unknown") || !strings.Contains(string(body), "build_user") {
		t.Errorf("Cannot find 'unknown' or 'build_user' in /info content")
	}

	/* Test GET /404, additional handler */
	body, err = testGet(host+"/404", http.StatusNotFound)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !strings.Contains(string(body), "404 page not found") {
		t.Errorf("Cannot find 'Not found' in /404 content: %s", string(body))
	}

	cancel()

}

// testGet is a smaller helper to query an URL and return its content
func testGet(url string, expectedStatus int) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, fmt.Errorf("Cannot query %s: %s", url, err.Error())
	}
	if resp.StatusCode != expectedStatus {
		return []byte{}, fmt.Errorf("Expecting %d HTTP status code on %s got %d", expectedStatus, url, resp.StatusCode)
	}
	//nolint:errcheck
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)

}
