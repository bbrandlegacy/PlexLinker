package aria2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func testClient(t *testing.T, body string, inspect func(rpcRequest)) *Client {
	t.Helper()
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var rpcReq rpcRequest
		if err := json.NewDecoder(request.Body).Decode(&rpcReq); err != nil {
			t.Fatal(err)
		}
		if inspect != nil {
			inspect(rpcReq)
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	return NewClient("http://aria2.test/jsonrpc", "test-secret", WithHTTPClient(httpClient))
}

func TestAddURIUsesTokenAndOptions(t *testing.T) {
	t.Parallel()
	client := testClient(t, `{"jsonrpc":"2.0","id":"1","result":"2089b05ecca3d829"}`, func(request rpcRequest) {
		if request.Method != "aria2.addUri" {
			t.Fatalf("method = %q", request.Method)
		}
		if got := request.Params[0]; got != "token:test-secret" {
			t.Fatalf("token = %#v", got)
		}
	})
	gid, err := client.AddURI(context.Background(), "https://example.test/video.mkv", AddOptions{"dir": "/media"})
	if err != nil {
		t.Fatal(err)
	}
	if gid != "2089b05ecca3d829" {
		t.Fatalf("gid = %q", gid)
	}
}

func TestRPCError(t *testing.T) {
	t.Parallel()
	client := testClient(t, `{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"unauthorized"}}`, nil)
	_, err := client.Version(context.Background())
	if err == nil {
		t.Fatal("expected RPC error")
	}
	if rpcErr, ok := err.(*RPCError); !ok || rpcErr.Code != 1 {
		t.Fatalf("error = %#v", err)
	}
}

func TestTellActiveDecodesJobs(t *testing.T) {
	t.Parallel()
	client := testClient(t, `{"jsonrpc":"2.0","id":"1","result":[{"gid":"abc","status":"active","totalLength":"100","completedLength":"25","downloadSpeed":"10","files":[{"index":"1","path":"/media/file.mkv","length":"100","completedLength":"25"}]}]}`, nil)
	jobs, err := client.TellActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].GID != "abc" || jobs[0].Files[0].Path != "/media/file.mkv" {
		t.Fatalf("jobs = %#v", jobs)
	}
}

func TestRemoveDownloadResultUsesStoppedResultMethod(t *testing.T) {
	t.Parallel()
	client := testClient(t, `{"jsonrpc":"2.0","id":"1","result":"OK"}`, func(request rpcRequest) {
		if request.Method != "aria2.removeDownloadResult" {
			t.Fatalf("method = %q", request.Method)
		}
		if len(request.Params) != 2 || request.Params[1] != "failed-gid" {
			t.Fatalf("params = %#v", request.Params)
		}
	})
	if err := client.RemoveDownloadResult(context.Background(), "failed-gid"); err != nil {
		t.Fatal(err)
	}
}
