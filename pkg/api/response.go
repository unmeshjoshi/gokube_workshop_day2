package api

import (
	"log"

	"github.com/emicklei/go-restful/v3"
)

// WriteResponse is a helper function to write the response and log any errors
func WriteResponse(response *restful.Response, status int, entity interface{}) {
	if entity != nil {
		if err := response.WriteHeaderAndEntity(status, entity); err != nil {
			log.Printf("Error writing response: %v", err)
		}
		return
	}

	response.WriteHeader(status)
}

// WriteError is a helper function to write an error response
func WriteError(response *restful.Response, status int, err error) {
	if writeErr := response.WriteError(status, err); writeErr != nil {
		log.Printf("Error writing error response: %v", writeErr)
	}
}
