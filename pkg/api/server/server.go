package server

import (
	"net/http"

	"gokube/pkg/api"
	"gokube/pkg/api/handlers"
	"gokube/pkg/registry"

	"github.com/emicklei/go-restful/v3"

	"gokube/pkg/storage"
)

// APIServer represents the API server
type APIServer struct {
	nodeRegistry       *registry.NodeRegistry
	podRegistry        *registry.PodRegistry
	replicasetRegistry *registry.ReplicaSetRegistry
}

// NewAPIServer creates a new instance of APIServer
func NewAPIServer(storage storage.Storage) *APIServer {
	return &APIServer{
		nodeRegistry:       registry.NewNodeRegistry(storage),
		podRegistry:        registry.NewPodRegistry(storage),
		replicasetRegistry: registry.NewReplicaSetRegistry(storage),
	}
}

// Start initializes and starts the API server
func (s *APIServer) Start(address string) error {
	container := restful.NewContainer()
	s.registerRoutes(container)

	return http.ListenAndServe(address, container)
}

// registerRoutes adds routes to the container
func (s *APIServer) registerRoutes(container *restful.Container) {
	ws := new(restful.WebService)

	ws.Path("/api/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/healthz").To(s.healthz))
	handlers.RegisterPodRoutes(ws, handlers.NewPodHandler(s.podRegistry))
	handlers.RegisterNodeRoutes(ws, handlers.NewNodeHandler(s.nodeRegistry))
	handlers.RegisterReplicasetRoutes(ws, handlers.NewReplicasetHandler(s.replicasetRegistry))

	container.Add(ws)
}

func (s *APIServer) healthz(request *restful.Request, response *restful.Response) {
	api.WriteResponse(response, http.StatusOK, nil)
}
