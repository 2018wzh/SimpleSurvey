package response

import "github.com/gin-gonic/gin"

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func JSON(c *gin.Context, statusCode int, code int, message string, data interface{}, meta interface{}) {
	c.JSON(statusCode, Body{
		Code:    code,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, 200, 200, "success", data, nil)
}

func Created(c *gin.Context, message string, data interface{}) {
	JSON(c, 201, 201, message, data, nil)
}

func Error(c *gin.Context, statusCode int, code int, message string, details interface{}) {
	c.JSON(statusCode, Body{
		Code:    code,
		Message: message,
		Details: details,
	})
}
