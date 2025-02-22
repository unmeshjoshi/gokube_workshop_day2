package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"gokube/pkg/api"
	"gokube/pkg/registry"

	"github.com/emicklei/go-restful/v3"
)

// NodeHandler handles Node-related HTTP requests
type NodeHandler struct {
	nodeRegistry *registry.NodeRegistry
}

// NewNodeHandler creates a new NodeHandler
func NewNodeHandler(nodeRegistry *registry.NodeRegistry) *NodeHandler {
	return &NodeHandler{nodeRegistry: nodeRegistry}
}

const nodeAttributeKey = "node"

// LoadNodeIntoRequest retrieves the node and stores it in the request attributes
func (h *NodeHandler) LoadNodeIntoRequest(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	name := req.PathParameter("name")
	node, err := h.nodeRegistry.GetNode(req.Request.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeNotFound):
			api.WriteError(resp, http.StatusNotFound, err)
		default:
			api.WriteError(resp, http.StatusInternalServerError, err)
		}
		return
	}
	req.SetAttribute(nodeAttributeKey, node)
	chain.ProcessFilter(req, resp)
}

// CreateNode handles POST requests to create a new Node
func (h *NodeHandler) CreateNode(request *restful.Request, response *restful.Response) {
	node := new(api.Node)
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if err := h.nodeRegistry.CreateNode(request.Request.Context(), node); err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeAlreadyExists):
			api.WriteError(response, http.StatusConflict, err)
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusCreated, node)
}

// GetNode handles GET requests to retrieve a Node
func (h *NodeHandler) GetNode(request *restful.Request, response *restful.Response) {
	node, ok := request.Attribute(nodeAttributeKey).(*api.Node)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve node from request attributes"))
		return
	}
	api.WriteResponse(response, http.StatusOK, node)
}

// UpdateNode handles PUT requests to update a Node
func (h *NodeHandler) UpdateNode(request *restful.Request, response *restful.Response) {
	existingNode, ok := request.Attribute(nodeAttributeKey).(*api.Node)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve node from request attributes"))
		return
	}

	node := new(api.Node)
	if err := request.ReadEntity(node); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if existingNode.Name != node.Name {
		api.WriteError(response, http.StatusBadRequest, fmt.Errorf("node name in URL does not match the name in the request body"))
		return
	}

	if err := h.nodeRegistry.UpdateNode(request.Request.Context(), node); err != nil {
		switch {
		case errors.Is(err, registry.ErrNodeInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
		}
		return
	}

	api.WriteResponse(response, http.StatusOK, node)
}

// DeleteNode handles DELETE requests to remove a Node
func (h *NodeHandler) DeleteNode(request *restful.Request, response *restful.Response) {
	node, ok := request.Attribute(nodeAttributeKey).(*api.Node)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve node from request attributes"))
		return
	}

	if err := h.nodeRegistry.DeleteNode(request.Request.Context(), node.Name); err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusNoContent, nil)
}

// ListNodes handles GET requests to list all Nodes
func (h *NodeHandler) ListNodes(request *restful.Request, response *restful.Response) {

	nodeName := request.Attribute("nodeName")
	nodes, err := h.nodeRegistry.ListNodes(request.Request.Context())
	if err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}
	if nodeName != nil {

	}

	api.WriteResponse(response, http.StatusOK, nodes)
}

// RegisterNodeRoutes registers Node routes with the WebService
func RegisterNodeRoutes(ws *restful.WebService, handler *NodeHandler) {
	ws.Route(ws.POST("/nodes").To(handler.CreateNode))
	ws.Route(ws.GET("/nodes").To(handler.ListNodes))
	ws.Route(ws.GET("/nodes/{name}").Filter(handler.LoadNodeIntoRequest).To(handler.GetNode))
	ws.Route(ws.PUT("/nodes/{name}").Filter(handler.LoadNodeIntoRequest).To(handler.UpdateNode))
	ws.Route(ws.DELETE("/nodes/{name}").Filter(handler.LoadNodeIntoRequest).To(handler.DeleteNode))
}
