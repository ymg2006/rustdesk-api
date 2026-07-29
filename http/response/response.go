package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
type PageData struct {
	Page  int         `json:"page"`
	Total int         `json:"total"`
	List  interface{} `json:"list"`
}

type DataResponse struct {
	Total uint        `json:"total"`
	Data  interface{} `json:"data"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func SendResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		code, message, data,
	})
}

func Success(c *gin.Context, data interface{}) {
	SendResponse(c, 0, "success", data)
}

// Fail returns a failure response using the site-wide convention: HTTP 200 with business code/message/data in JSON.
//
// Security hardening for unified 500 responses: when code >= 500, never echo internal error details
// to clients, regardless of the caller-provided message. Such details may include SQL, table names,
// file paths, stack traces, or other sensitive information. Return only a generic message and log the
// original message server-side for troubleshooting. Client errors such as 4xx still pass messages through.
func Fail(c *gin.Context, code int, message string) {
	if code >= 500 {
		if message != "" {
			// Log only on the server side; never echo it to clients.
			global.Logger.Error("server error response suppressed for client: " + message)
		}
		message = TranslateMsg(c, "ServerInternalError")
	}
	SendResponse(c, code, message, nil)
}

func Error(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
	})
}

// ServerError returns the generic 5xx envelope (HTTP 200 + business code 500 + message).
// It avoids leaking internal details such as SQL or stack traces. Controllers should use it for
// unexpected server-side errors instead of returning err.Error() directly to clients.
func ServerError(c *gin.Context) {
	Fail(c, 500, "")
}

type ServerConfigResponse struct {
	IdServer    string `json:"id_server"`
	Key         string `json:"key"`
	RelayServer string `json:"relay_server"`
	ApiServer   string `json:"api_server"`
}

func TranslateMsg(c *gin.Context, messageId string) string {
	localizer := global.Localizer(c.GetHeader("Accept-Language"))
	errMsg, err := localizer.LocalizeMessage(&i18n.Message{
		ID: messageId,
	})
	if err != nil {
		global.Logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
func TranslateTempMsg(c *gin.Context, messageId string, templateData map[string]interface{}) string {
	localizer := global.Localizer(c.GetHeader("Accept-Language"))
	errMsg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
		TemplateData: templateData,
	})
	if err != nil {
		global.Logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
func TranslateParamMsg(c *gin.Context, messageId string, params ...string) string {
	localizer := global.Localizer(c.GetHeader("Accept-Language"))
	templateData := make(map[string]interface{})
	for i, v := range params {
		k := fmt.Sprintf("P%d", i)
		templateData[k] = v
	}
	errMsg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
		TemplateData: templateData,
	})
	if err != nil {
		global.Logger.Warn("LocalizeMessage Error: " + err.Error())
		errMsg = messageId
	}
	return errMsg
}
