package main

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"strconv"
	"time"

	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/joelanford/deppy-client-go/api"
	"github.com/joelanford/deppy-client-go/examples/internal/util"
	"github.com/joelanford/deppy-client-go/registry"
)

func main() {
	var (
		listenAddr   string
		cacheTimeout time.Duration
	)
	flag.StringVar(&listenAddr, "listen-addr", "localhost:8080", "Listen address of entity service.")
	flag.DurationVar(&cacheTimeout, "cache-timeout", 0, "Time after which cached kubernetes version information is considered stale. A value of 0 means the version information will be requested exactly once ever.")
	flag.Parse()

	l := util.FatalLogr{Logger: zap.New().WithName("kubernetes-cluster-entity-source")}

	reg := registry.New()
	if err := updateEntity(reg); err != nil {
		l.Fatal(err, "update kubernetes cluster entity")
	}
	lastPullTime := time.Now()

	l.Info("starting http server", "listen-addr", listenAddr)
	if err := http.ListenAndServe(listenAddr, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if cacheTimeout != 0 && time.Now().Sub(lastPullTime) > cacheTimeout {
			if err := updateEntity(reg); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			l.Info("refreshed cluster version information")
			lastPullTime = time.Now()
		}
		reg.Handler().ServeHTTP(writer, request)
	})); err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.Fatal(err, "http server failed")
	}
}

func updateEntity(reg *registry.Registry) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	cl, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}

	info, err := cl.ServerVersion()
	if err != nil {
		return err
	}

	infoJSON, err := json.Marshal(info)
	if err != nil {
		return err
	}

	entity := &api.Entity{
		ID:   "kubernetes-cluster",
		Data: infoJSON,
		Properties: []api.TypeValue{
			{Type: "git.tag", Value: []byte(strconv.Quote(info.GitVersion))},
			{Type: "git.commit", Value: []byte(strconv.Quote(info.GitCommit))},
			{Type: "semver.majorVersion", Value: []byte(info.Major)},
			{Type: "semver.minorVersion", Value: []byte(info.Minor)},
		},
	}

	return reg.Upsert(entity)
}
