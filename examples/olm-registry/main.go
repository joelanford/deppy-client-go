package main

import (
	"deppy-client-go/api"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/blang/semver/v4"
	"log"
	"net/http"

	"bytes"
	"github.com/operator-framework/operator-registry/pkg/client"
	"k8s.io/apimachinery/pkg/util/rand"
	"strconv"
)

func main() {
	var (
		registryAddr string
		listenAddr   string
	)

	flag.StringVar(&registryAddr, "registry-addr", "localhost:50051", "Address of the OLM registry GRPC service.")
	flag.StringVar(&listenAddr, "listen-addr", "localhost:8080", "Listen address of entity service.")

	log.Printf("connecting to OLM api.Registry GRPC service at %q...", registryAddr)
	grpcClient, err := client.NewClient(registryAddr)
	if err != nil {
		log.Fatal(err)
	}

	idPrefix := rand.String(16)

	log.Printf("listening on %q...", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		iter, err := grpcClient.ListBundles(request.Context())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		type message struct {
			Entity *api.Entity `json:"entity,omitempty"`
			Error  string      `json:"error,omitempty"`
		}

		enc := json.NewEncoder(writer)
		enc.SetEscapeHTML(false)
	iter:
		for {
			b := iter.Next()
			if b == nil {
				break
			}

			props := b.Properties
			b.Properties = nil
			b.CsvJson = ""
			b.Dependencies = nil
			b.ProvidedApis = nil
			b.Object = nil
			b.RequiredApis = nil
			bData, err := json.Marshal(b)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			e := &api.Entity{
				ID:   fmt.Sprintf("%s-%s", idPrefix, b.CsvName),
				Data: bData,
			}

			properties := make([]api.TypeValue, 0)
			constraints := make([]api.TypeValue, 0)

			for _, p := range props {
				switch p.Type {
				case "olm.package.required", "olm.gvk.required":
					constraints = append(constraints, api.TypeValue{
						Type:  p.Type,
						Value: []byte(p.Value),
					})
				case "olm.maxOpenShiftVersion":
					maxRange, err := convertMaxOpenshiftVersionToRange([]byte(p.Value))
					if err != nil {
						msg := message{Entity: e, Error: fmt.Sprintf("invalid bundle %q: %v", b.CsvName, err)}
						_ = enc.Encode(msg)
						continue iter
					}
					constraints = append(constraints, api.TypeValue{
						Type:  "olm.openshiftVersionRange",
						Value: maxRange,
					})
				default:
					properties = append(properties, api.TypeValue{
						Type:  p.Type,
						Value: []byte(p.Value),
					})
				}
				e.Properties = properties
				e.Constraints = constraints
			}

			msg := message{Entity: e}
			if err := enc.Encode(msg); err != nil {
				msg = message{Error: fmt.Sprintf("encode entity %q: %v", e.ID, err)}
				_ = enc.Encode(msg)
				continue iter
			}
		}
		if iter.Error() != nil {
			http.Error(writer, iter.Error().Error(), http.StatusInternalServerError)
			return
		}
	})))
}

func convertMaxOpenshiftVersionToRange(value []byte) ([]byte, error) {
	if !bytes.HasPrefix(value, []byte(`"`)) {
		value = []byte(strconv.Quote(string(value)))
	}

	var iface interface{}
	if err := json.Unmarshal([]byte(value), &iface); err != nil {
		return nil, fmt.Errorf("unmarshal maxOpenShiftVersion JSON %q: %v", string(value), err)
	}

	verStr, ok := iface.(string)
	if !ok {
		return nil, fmt.Errorf("parse maxOpenShiftVersion %[1]q: invalid type %[1]T", iface)
	}

	vers, err := semver.ParseTolerant(verStr)
	if err != nil {
		return nil, fmt.Errorf("parse maxOpenShiftVersion %q: %v", verStr, err)
	}
	return json.Marshal(fmt.Sprintf("<=%d.%d.x", vers.Major, vers.Minor))
}
