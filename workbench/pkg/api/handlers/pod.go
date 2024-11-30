package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"gokube/pkg/api"
	"gokube/pkg/registry"
)

// PodHandler handles Pod-related requests
type PodHandler struct {
	podRegistry *registry.PodRegistry
}

// NewPodHandler creates a new instance of PodHandler
func NewPodHandler(podRegistry *registry.PodRegistry) *PodHandler {
	return &PodHandler{podRegistry: podRegistry}
}

const podAttributeKey = "pod"

// LoadPodIntoRequest retrieves the pod and stores it in the request attributes
func (h *PodHandler) LoadPodIntoRequest(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	name := req.PathParameter("name")
	pod, err := h.podRegistry.GetPod(req.Request.Context(), name)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrPodNotFound):
			api.WriteError(resp, http.StatusNotFound, err)
		default:
			api.WriteError(resp, http.StatusInternalServerError, err)
		}
		return
	}
	req.SetAttribute(podAttributeKey, pod)
	chain.ProcessFilter(req, resp)
}

// CreatePod handles POST requests to create a new Pod
func (h *PodHandler) CreatePod(request *restful.Request, response *restful.Response) {
	pod := new(api.Pod)
	if err := request.ReadEntity(pod); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if err := h.podRegistry.CreatePod(request.Request.Context(), pod); err != nil {
		switch {
		case errors.Is(err, registry.ErrPodAlreadyExists):
			api.WriteError(response, http.StatusConflict, err)
		case errors.Is(err, registry.ErrPodInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
			return
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
			return
		}
	}

	api.WriteResponse(response, http.StatusCreated, pod)
}

// ListPods handles GET requests to list all Pods
func (h *PodHandler) ListPods(request *restful.Request, response *restful.Response) {
	pods, err := h.podRegistry.ListPods(request.Request.Context())
	if err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusOK, pods)
}

// GetPod handles GET requests to retrieve a Pod
func (h *PodHandler) GetPod(request *restful.Request, response *restful.Response) {
	pod, ok := request.Attribute(podAttributeKey).(*api.Pod)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve pod from request attributes"))
		return
	}
	api.WriteResponse(response, http.StatusOK, pod)
}

// UpdatePod handles PUT requests to update a Pod
func (h *PodHandler) UpdatePod(request *restful.Request, response *restful.Response) {
	existingPod, ok := request.Attribute(podAttributeKey).(*api.Pod)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve pod from request attributes"))
		return
	}

	updatedPod := new(api.Pod)
	if err := request.ReadEntity(updatedPod); err != nil {
		api.WriteError(response, http.StatusBadRequest, err)
		return
	}

	if existingPod.Name != updatedPod.Name {
		api.WriteError(response, http.StatusBadRequest, fmt.Errorf("pod name in URL does not match pod name in request body"))
		return
	}

	if err := h.podRegistry.UpdatePod(request.Request.Context(), updatedPod); err != nil {
		switch {
		case errors.Is(err, registry.ErrPodInvalid):
			api.WriteError(response, http.StatusBadRequest, err)
			return
		default:
			api.WriteError(response, http.StatusInternalServerError, err)
			return
		}
	}

	api.WriteResponse(response, http.StatusOK, updatedPod)
}

// DeletePod handles DELETE requests to remove a Pod
func (h *PodHandler) DeletePod(request *restful.Request, response *restful.Response) {
	pod, ok := request.Attribute(podAttributeKey).(*api.Pod)
	if !ok {
		api.WriteError(response, http.StatusInternalServerError, fmt.Errorf("failed to retrieve pod from request attributes"))
		return
	}

	if err := h.podRegistry.DeletePod(request.Request.Context(), pod.Name); err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusNoContent, nil)
}

// ListUnassignedPods handles GET requests to list all unassigned Pods
func (h *PodHandler) ListUnassignedPods(request *restful.Request, response *restful.Response) {
	pods, err := h.podRegistry.ListUnassignedPods(request.Request.Context())
	if err != nil {
		api.WriteError(response, http.StatusInternalServerError, err)
		return
	}

	api.WriteResponse(response, http.StatusOK, pods)
}

func RegisterPodRoutes(ws *restful.WebService, podHandler *PodHandler) {
	ws.Route(ws.POST("/pods").To(podHandler.CreatePod))
	ws.Route(ws.GET("/pods").To(podHandler.ListPods))
	ws.Route(ws.GET("/pods/{name}").Filter(podHandler.LoadPodIntoRequest).To(podHandler.GetPod))
	ws.Route(ws.PUT("/pods/{name}").Filter(podHandler.LoadPodIntoRequest).To(podHandler.UpdatePod))
	ws.Route(ws.DELETE("/pods/{name}").Filter(podHandler.LoadPodIntoRequest).To(podHandler.DeletePod))
	ws.Route(ws.GET("/pods/unassigned").To(podHandler.ListUnassignedPods))
}
