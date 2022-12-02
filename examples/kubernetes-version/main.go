package main

import (
	"deppy-client-go/api"
	"deppy-client-go/registry"
	"encoding/json"
	"flag"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strconv"
)

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen-addr", "localhost:8080", "Listen address of entity service.")

	cfg := config.GetConfigOrDie()
	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	info, err := cl.Discovery().ServerVersion()
	if err != nil {
		log.Fatal(err)
	}

	infoJSON, err := json.Marshal(info)
	if err != nil {
		log.Fatal(err)
	}

	entity := &api.Entity{
		ID:   "cluster",
		Data: infoJSON,
		Properties: []api.TypeValue{
			{Type: "git.tag", Value: []byte(strconv.Quote(info.GitVersion))},
			{Type: "git.commit", Value: []byte(strconv.Quote(info.GitCommit))},
			{Type: "semver.majorVersion", Value: []byte(info.Major)},
			{Type: "semver.minorVersion", Value: []byte(info.Minor)},
		},
	}
	entityJSON, err := json.Marshal(entity)
	if err != nil {
		log.Fatal(err)
	}

	reg := registry.New()
	reg.MustUpsert(entity)

	log.Printf("serving kubernetes cluster entity: %s", string(entityJSON))
	log.Printf("listening on %q...", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, reg.Handler()))
}
