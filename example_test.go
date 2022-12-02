package deppy_client_go_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/joelanford/deppy-client-go/api"
	"github.com/joelanford/deppy-client-go/registry"
)

func ExampleHelloWorld() {
	reg := registry.New()
	reg.MustUpsert(&api.Entity{
		ID:   "hello",
		Data: []byte(`"world"`),
		Properties: []api.TypeValue{
			{
				Type:  "isGreeting",
				Value: []byte(`true`),
			},
		},
	})

	server := httptest.NewServer(reg.Handler())
	defer server.Close()

	printEntities(server)
	// Output: {"id":"hello","data":"world","properties":[{"type":"isGreeting","value":true}]}
}

func printEntities(server *httptest.Server) {
	resp, err := http.Get(server.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("unexpected error %s: %v", resp.Status, string(data))
	}

	fmt.Println(string(data))
}
