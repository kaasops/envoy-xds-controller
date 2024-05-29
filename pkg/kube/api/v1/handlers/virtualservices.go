package handlers

import (
	"fmt"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/kube/api/v1/types"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gin-gonic/gin"
)

type VirtualServiceHandler struct {
	Client client.Client
	Config *config.Config
}

func NewVirtualServiceHandler(client client.Client, config *config.Config) *VirtualServiceHandler {
	return &VirtualServiceHandler{Client: client, Config: config}
}

// GetAllVirtualServices gets the names of all Virtual Services.
// @Summary Get all Virtual Services
// @Description Get names of all Virtual Services
// @Tags virtualservices
// @Produce json
// @Param namespace query string false "Namespace"
// @Success 200 {array} string
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices [get]
func (h *VirtualServiceHandler) GetAllVirtualServices(c *gin.Context) {
	namespace, err := h.getNamespace(c)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	vs := v1alpha1.VirtualService{}
	names, err := vs.GetAll(c, namespace, h.Client)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, names)
}

// GetAllVirtualServicesWithWrongState gets the names of all Virtual Services with an erroneous state.
// @Summary Get all Virtual Services with erroneous state
// @Description Get names of all Virtual Services that have an erroneous state
// @Tags virtualservices
// @Produce json
// @Param namespace query string false "Namespace"
// @Success 200 {array} string
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices/wrong-state [get]
func (h *VirtualServiceHandler) GetAllVirtualServicesWithWrongState(c *gin.Context) {
	namespace, err := h.getNamespace(c)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	vs := v1alpha1.VirtualService{}
	names, err := vs.GetAllWithWrongState(c, namespace, h.Client)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, names)
}

// GetVirtualService gets a specific Virtual Service by name.
// @Summary Get a specific Virtual Service
// @Description Get a specific Virtual Service by name
// @Tags virtualservices
// @Produce json
// @Param name path string true "Virtual Service Name"
// @Param namespace query string false "Namespace"
// @Success 200 {object} v1alpha1.VirtualService
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices/{name} [get]
func (h *VirtualServiceHandler) GetVirtualService(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respondWithError(c, http.StatusBadRequest, "Name is required")
		return
	}

	namespace, err := h.getNamespace(c)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	vs := v1alpha1.VirtualService{}
	item, err := vs.Get(c, name, namespace, h.Client)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// GetVirtualServiceByNameAndNodeId searches for a Virtual Service by name and nodeID.
// @Summary Search for a Virtual Service by name and nodeID
// @Description Search in which Virtual Service a specific domain is described
// @Tags virtualservices
// @Produce json
// @Param name query string true "Virtual Service Name"
// @Param nodeId query string true "Node ID"
// @Param namespace query string false "Namespace"
// @Success 200 {object} v1alpha1.VirtualService
// @Failure 400 {object} types.ErrorResponse
// @Failure 404 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices/search [get]
func (h *VirtualServiceHandler) GetVirtualServiceByNameAndNodeId(c *gin.Context) {
	name := c.Query("name")
	nodeId := c.Query("nodeId")

	if name == "" || nodeId == "" {
		respondWithError(c, http.StatusBadRequest, "Name and nodeId are required")
		return
	}

	namespace, err := h.getNamespace(c)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	vs := v1alpha1.VirtualService{}
	item, err := vs.GetByNameAndNodeId(c, name, nodeId, namespace, h.Client)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// CreateVirtualService creates a new Virtual Service.
// @Summary Create a new Virtual Service
// @Description Create a new Virtual Service
// @Tags virtualservices
// @Accept json
// @Produce json
// @Param virtualservice body v1alpha1.VirtualService true "Virtual Service object"
// @Success 201 {object} v1alpha1.VirtualService
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices [post]
func (h *VirtualServiceHandler) CreateVirtualService(c *gin.Context) {
	var vs v1alpha1.VirtualService
	if err := c.ShouldBindJSON(&vs); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Ensure the name is provided
	if vs.Name == "" {
		respondWithError(c, http.StatusBadRequest, "Name is required")
		return
	}

	if vs.Namespace == "" {
		if h.Config.WatchNamespace != "" {
			vs.Namespace = h.Config.WatchNamespace
		} else {
			respondWithError(c, http.StatusBadRequest, "Namespace is required")
			return
		}
	}

	s := v1alpha1.VirtualService{}
	if err := s.CreateVirtualService(c, &vs, h.Client); err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, vs)
}

// UpdateVirtualService updates an existing Virtual Service.
// @Summary Update an existing Virtual Service
// @Description Update an existing Virtual Service
// @Tags virtualservices
// @Accept json
// @Produce json
// @Param virtualservice body v1alpha1.VirtualService true "Virtual Service object"
// @Success 200 {object} v1alpha1.VirtualService
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices [put]
func (h *VirtualServiceHandler) UpdateVirtualService(c *gin.Context) {
	var vs v1alpha1.VirtualService
	if err := c.ShouldBindJSON(&vs); err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Ensure the name is provided
	if vs.Name == "" {
		respondWithError(c, http.StatusBadRequest, "Name is required")
		return
	}

	// Default to "default" namespace if not provided
	if vs.Namespace == "" {
		if h.Config.WatchNamespace != "" {
			vs.Namespace = h.Config.WatchNamespace
		} else {
			respondWithError(c, http.StatusBadRequest, "Namespace is required")
			return
		}
	}

	s := v1alpha1.VirtualService{}
	if err := s.UpdateVirtualService(c, &vs, h.Client); err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, vs)
}

// DeleteVirtualService deletes a Virtual Service.
// @Summary Delete a Virtual Service
// @Description Delete a Virtual Service
// @Tags virtualservices
// @Produce json
// @Param name path string true "Virtual Service Name"
// @Param namespace query string false "Namespace"
// @Success 204
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /virtualservices/{name} [delete]
func (h *VirtualServiceHandler) DeleteVirtualService(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		respondWithError(c, http.StatusBadRequest, "Name is required")
		return
	}

	namespace, err := h.getNamespace(c)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	s := v1alpha1.VirtualService{}
	if err := s.DeleteVirtualService(c, name, namespace, h.Client); err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *VirtualServiceHandler) getNamespace(c *gin.Context) (string, error) {
	namespace := c.Query("namespace")
	if namespace == "" && h.Config.WatchNamespace == "" {
		return "", fmt.Errorf("namespace is required")
	}
	if namespace == "" {
		namespace = h.Config.WatchNamespace
	}
	return namespace, nil
}

func respondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, types.ErrorResponse{
		Code:    code,
		Message: message,
	})
}
