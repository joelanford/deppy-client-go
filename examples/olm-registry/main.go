package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"

	"github.com/blang/semver/v4"
	"github.com/go-logr/logr"
	api2 "github.com/operator-framework/operator-registry/pkg/api"
	"github.com/operator-framework/operator-registry/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/joelanford/deppy-client-go/api"
	"github.com/joelanford/deppy-client-go/examples/internal/util"
)

func main() {
	var (
		registryAddr string
		listenAddr   string
	)

	flag.StringVar(&registryAddr, "registry-addr", "localhost:50051", "Address of the OLM registry GRPC service.")
	flag.StringVar(&listenAddr, "listen-addr", "localhost:8080", "Listen address of entity service.")
	flag.Parse()

	l := util.FatalLogr{Logger: zap.New().WithName("catalog-source-adapter")}
	l.Info("connecting to api.Registry service", "registry-addr", registryAddr)
	grpcClient, err := client.NewClient(registryAddr)
	if err != nil {
		l.Fatal(err, "create grpc client")
	}

	l.Info("starting http server", "listen-addr", listenAddr)
	if err := http.ListenAndServe(listenAddr, handler(l.Logger, grpcClient)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.Fatal(err, "http server failed")
	}
}

func handler(l logr.Logger, grpcClient *client.Client) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		iter, err := grpcClient.ListBundles(request.Context())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			l.Error(err, "grpc api.Registry/ListBundles: %v", err)
			return
		}
		enc := json.NewEncoder(writer)
		enc.SetEscapeHTML(false)
	loop:
		for {
			b := iter.Next()
			if b == nil {
				break
			}

			e, err := bundleToEntity(b)
			if err != nil {
				l.Error(err, "skipping invalid bundle", "package", b.PackageName, "channel", b.ChannelName, "bundle", b.BundlePath)
				continue loop
			}

			if err := enc.Encode(e); err != nil {
				l.Error(err, "error encoding entity for bundle", "package", b.PackageName, "channel", b.ChannelName, "bundle", b.BundlePath)
				continue loop
			}
		}
		if iter.Error() != nil {
			l.Error(err, "error iterating bundles")
			return
		}
	})
}

func bundleToEntity(b *api2.Bundle) (*api.Entity, error) {
	properties := make([]api.TypeValue, 0)
	constraints := make([]api.TypeValue, 0)

	for _, p := range b.Properties {
		switch p.Type {
		case "olm.package.required", "olm.gvk.required":
			constraints = append(constraints, api.TypeValue{
				Type:  p.Type,
				Value: []byte(p.Value),
			})
		case "olm.maxOpenShiftVersion":
			maxRange, err := convertMaxOpenshiftVersionToRange([]byte(p.Value))
			if err != nil {
				return nil, err
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
	}
	b.Properties = nil
	b.CsvJson = ""
	b.Dependencies = nil
	b.ProvidedApis = nil
	b.Object = nil
	b.RequiredApis = nil
	bData, err := marshalJSON(b)
	if err != nil {
		return nil, err
	}
	return &api.Entity{
		ID:          b.BundlePath,
		Data:        bData,
		Properties:  properties,
		Constraints: constraints,
	}, nil
}

func convertMaxOpenshiftVersionToRange(value []byte) ([]byte, error) {
	origValue := value
	if !bytes.HasPrefix(value, []byte(`"`)) {
		value = []byte(strconv.Quote(string(value)))
	}

	var iface interface{}
	if err := json.Unmarshal(value, &iface); err != nil {
		return nil, fmt.Errorf("unmarshal maxOpenShiftVersion JSON %q: %v", string(origValue), err)
	}

	verStr, ok := iface.(string)
	if !ok {
		return nil, fmt.Errorf("parse maxOpenShiftVersion %[1]q: invalid type %[1]T", iface)
	}

	vers, err := semver.ParseTolerant(verStr)
	if err != nil {
		return nil, fmt.Errorf("parse maxOpenShiftVersion %q: %v", verStr, err)
	}

	maxVers := fmt.Sprintf("<=%d.%d.x", vers.Major, vers.Minor)
	out, err := marshalJSON(maxVers)
	if err != nil {
		return nil, fmt.Errorf("encode maxOpenShiftVersion %q: %v", maxVers, err)
	}
	return out, nil
}

func marshalJSON(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
