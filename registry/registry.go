package registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"github.com/go-logr/logr"

	"github.com/joelanford/deppy-client-go/api"
	"github.com/joelanford/deppy-client-go/internal/util"
)

type Registry struct {
	entities util.SyncMap[string, *api.Entity]
	Log      logr.Logger
}

func New() *Registry {
	return &Registry{entities: util.NewSyncMap[string, *api.Entity]()}
}

func (r *Registry) MustUpsert(e *api.Entity) {
	if err := r.Upsert(e); err != nil {
		panic(err)
	}
}

func (r *Registry) Upsert(e *api.Entity) error {
	if e == nil {
		return errors.New("nil entity cannot be upserted")
	}
	if e.ID == "" {
		return errors.New("entity must have an ID")
	}
	r.entities.Set(e.ID, e)
	return nil
}

func (r *Registry) Delete(id string) {
	r.entities.Delete(id)
}

func (r *Registry) Handler() http.Handler {
	entities := r.entities.Values()
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].ID < entities[j].ID
	})

	return http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(resp)
		enc.SetEscapeHTML(false)
		for _, e := range entities {
			if err := enc.Encode(e); err != nil {
				r.Log.Error(err, "error encoding entity", "entityID", e.ID)
				continue
			}
		}
	})
}
