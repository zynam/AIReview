package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type errorBody struct {
	Error errorMessage `json:"error"`
}

type errorMessage struct {
	Message string `json:"message"`
}

func writeError(c *gin.Context, status int, err error) {
	message := http.StatusText(status)
	if err != nil && err.Error() != "" {
		message = err.Error()
	}
	c.JSON(status, errorBody{
		Error: errorMessage{Message: message},
	})
}
