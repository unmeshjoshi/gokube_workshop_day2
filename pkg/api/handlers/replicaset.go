package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"gokube/pkg/api"
	"gokube/pkg/registry"

	"github.com/emicklei/go-restful/v3"
)

// ReplicasetHandler handles Replicaset-related HTTP requests
type ReplicasetHandler struct {
	replicasetRegistry *registry.ReplicaSetRegistry
}

// NewReplicasetHandler creates a new ReplicasetHandler
func NewReplicasetHandler(replicasetRegistry *registry.ReplicaSetRegistry) *ReplicasetHandler {
	return &ReplicasetHandler{replicasetRegistry: replicasetRegistry}
}

const replicasetAttributeKey = "replicaset"

// LoadReplicasetIntoRequest retrieves the replicaset and stores it in the request attributes
func (h *ReplicasetHandler) LoadReplicasetIntoRequest(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	name := req.PathParameter("name")
	replicaset, err := h.replicasetRegistry.Get(req.Request.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrReplicaSetNotFound):
			api.WriteError(resp, http.StatusNotFound, err)
		default:
			api.WriteError(resp, http.StatusInternalServerError, err)
		}
		return
	}
	req.SetAttribute(replicasetAttributeKey, replicaset)
	chain.ProcessFilter(req, resp)
}

// CreateReplicaset handles POST requests to create a new Replicaset
func (h *ReplicasetHandler) CreateReplicaset(request *restful.Request, response *restful.Response) {
	replicaset := new(api.ReplicaSet)
	if err := request.ReadEntity(replicaset); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if err := h.replicasetRegistry.Create(request.Request.Context(), replicaset); err != nil {
		switch {
		case errors.Is(err, registry.ErrReplicaSetExists):
			api.WriteError(response, http.StatusConflict, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusCreated, replicaset)
}

// GetReplicaset handles GET requests to retrieve a Replicaset
func (h *ReplicasetHandler) GetReplicaset(request *restful.Request, response *restful.Response) {
	replicaset, ok := request.Attribute(replicasetAttributeKey).(*api.ReplicaSet)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve replicaset from request attributes"))
		return
	}
	api.WriteResponse(response, http.StatusOK, replicaset)
}

// UpdateReplicaset handles PUT requests to update a replicaset
func (h *ReplicasetHandler) UpdateReplicaset(request *restful.Request, response *restful.Response) {
	existingReplicaset, ok := request.Attribute(replicasetAttributeKey).(*api.ReplicaSet)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve replicaset from request attributes"))
		return
	}

	replicaset := new(api.ReplicaSet)
	if err := request.ReadEntity(replicaset); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if existingReplicaset.Name != replicaset.Name {
		api.WriteError(response, http.StatusBadRequest, fmt.Errorf("replicaset name in URL does not match the replicaset in the request body"))
		return
	}

	if err := h.replicasetRegistry.Update(request.Request.Context(), replicaset); err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusOK, replicaset)
}

// DeleteReplicaset handles DELETE requests to remove a replicaset
func (h *ReplicasetHandler) DeleteReplicaset(request *restful.Request, response *restful.Response) {
	replicaset, ok := request.Attribute(replicasetAttributeKey).(*api.ReplicaSet)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve replicaset from request attributes"))
		return
	}

	if err := h.replicasetRegistry.Delete(request.Request.Context(), replicaset.Name); err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusNoContent, nil)
}

// ListReplicasets handles GET requests to list all replicasets
func (h *ReplicasetHandler) ListReplicasets(request *restful.Request, response *restful.Response) {
	replicasets, err := h.replicasetRegistry.List(request.Request.Context())
	if err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusOK, replicasets)
}

// RegisterReplicasetRoutes registers replicaset routes with the WebService
func RegisterReplicasetRoutes(ws *restful.WebService, handler *ReplicasetHandler) {
	ws.Route(ws.POST("/replicasets").To(handler.CreateReplicaset))
	ws.Route(ws.GET("/replicasets").To(handler.ListReplicasets))
	ws.Route(ws.GET("/replicasets/{name}").Filter(handler.LoadReplicasetIntoRequest).To(handler.GetReplicaset))
	ws.Route(ws.PUT("/replicasets/{name}").Filter(handler.LoadReplicasetIntoRequest).To(handler.UpdateReplicaset))
	ws.Route(ws.DELETE("/replicasets/{name}").Filter(handler.LoadReplicasetIntoRequest).To(handler.DeleteReplicaset))
}
