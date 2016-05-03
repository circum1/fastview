package main

import (
    "testing"
	"net/http"
    "net/http/httptest"
    "bytes"
    "fmt"
)

func TestLocalFSProvider_Path2Url(t *testing.T) {
    cp := &LocalFilesystemProvider{}
	var tests = []struct {
		input  string
		expected string
	}{
        {"others", "/image/others"},
        {"/home/fery/Pictures/something", "/image/something"},
        {"/home/fery/Pictures/somedir/something", "/image/somedir/something"},
    }
	for _, test := range tests {
        if result:=cp.Path2Url(test.input); result != test.expected {
            t.Errorf("Expected %v, got %v", test.expected, result)
        }
    }
}

func XTestServeDir(t *testing.T) {
    rw := httptest.NewRecorder()
    rw.Body = new(bytes.Buffer)
    req, _ := http.NewRequest("GET", "http://localhost:8080/image/others/", nil)

    serveLocal(rw, req)

    fmt.Printf("Result: %v\n", rw)

    if g, w := rw.Code, 200; g != w {
        t.Errorf("%s: code = %d, want %d", 200, g, w)
    }
    if g, w := rw.Body.String(), "x"; g != w {
        t.Errorf("%s: body = %v, want %q", "x", g, w)
    }
}
