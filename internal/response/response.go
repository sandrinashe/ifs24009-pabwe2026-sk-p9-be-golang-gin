// Package response menyeragamkan bentuk JSON seluruh endpoint: {status, message, data?}.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body adalah envelope standar.
type Body struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success membalas dengan status "success" (2xx).
func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Body{Status: "success", Message: message, Data: data})
}

// Fail membalas kesalahan klien (4xx) dengan status "fail".
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "fail", Message: message})
}

// Error membalas kesalahan server (5xx) dengan status "error".
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "error", Message: message})
}

// AbortFail seperti Fail tetapi menghentikan rantai middleware.
func AbortFail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "fail", Message: message})
}

// AbortError seperti Error tetapi menghentikan rantai middleware.
func AbortError(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, Body{Status: "error", Message: message})
}

// NotFound adalah handler NoRoute.
func NotFound(c *gin.Context) {
	Fail(c, http.StatusNotFound, "Endpoint not found")
}
