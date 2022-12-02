package registry

import (
	"deppy-client-go/api"
	"deppy-client-go/internal/util"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"net/http"
	"sort"
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

	type message struct {
		Entity *api.Entity `json:"entity,omitempty"`
		Error  string      `json:"error,omitempty"`
	}

	return http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
		enc := json.NewEncoder(resp)
		enc.SetEscapeHTML(false)
		for _, e := range entities {
			if err := enc.Encode(message{Entity: e}); err != nil {
				_ = enc.Encode(message{Error: fmt.Sprintf("failed to encode entity %q: %v", e.ID, err)})
				r.Log.Error(err, "encode entity", "entityID", e.ID)
				return
			}
		}
	})
}
