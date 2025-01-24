package main

import (
	"context"
	"fmt"
	"github.com/dreamph/handlex"
	"github.com/gofiber/fiber/v2"
	errs "github.com/pkg/errors"
	"io"
	"log"
	"mime/multipart"
	"time"
)

type Fiber struct {
	*fiber.Ctx
}

func (fcw *Fiber) Method() string {
	return fcw.Ctx.Method()
}

func (fcw *Fiber) UserContext() context.Context {
	return fcw.Ctx.UserContext()
}

func (fcw *Fiber) SendString(statusCode int, text string) error {
	return fcw.Ctx.Status(statusCode).SendString(text)
}

func (fcw *Fiber) SendStream(stream io.Reader, size ...int) error {
	return fcw.Ctx.SendStream(stream, size...)
}

func (fcw *Fiber) JSON(data interface{}) error {
	return fcw.Ctx.JSON(data)
}

func (fcw *Fiber) BodyParser(out interface{}) error {
	return fcw.Ctx.BodyParser(out)
}

func (fcw *Fiber) FormFile(key string) (*multipart.FileHeader, error) {
	return fcw.Ctx.FormFile(key)
}

func (fcw *Fiber) Get(key string, defaultValue ...string) string {
	return fcw.Ctx.Get(key, defaultValue...)
}

func (fcw *Fiber) Status(statusCode int) {
	fcw.Ctx.Status(statusCode)
}

type RequestInfo struct {
	Token string `json:"token"`
}

type RequestOption struct {
	EnableValidate bool
	SuccessStatus  int
}

func EnableValidate(enable bool) handlex.RequestOptions[RequestOption] {
	return func(opts *RequestOption) {
		opts.EnableValidate = enable
	}
}

func SuccessStatus(successStatus int) handlex.RequestOptions[RequestOption] {
	return func(opts *RequestOption) {
		opts.SuccessStatus = successStatus
	}
}

type ErrorResponse struct {
	Status        bool            `json:"status"`
	StatusCode    int             `json:"statusCode"`
	StatusMessage string          `json:"statusMessage"`
	Type          string          `json:"type"`
	Code          string          `json:"code"`
	Message       string          `json:"message"`
	ErrorMessage  string          `json:"errorMessage"`
	Time          time.Time       `json:"time" swaggertype:"string" format:"date-time"`
	Detail        string          `json:"detail"`
	ErrorData     *[]AppErrorData `json:"errorData"`
	Cause         error           `json:"-"`
}

type AppErrorData struct {
	Reference    string           `json:"reference"`
	ErrorDetails []AppErrorDetail `json:"errorDetails"`
}

type AppErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StreamData struct {
	Data io.Reader
	Size int `json:"status"`
}

type AppError struct {
	ErrCode    string `json:"errCode"`
	ErrMessage string `json:"errMessage"`
}

func (e *AppError) Error() string {
	return e.ErrCode + ":" + e.ErrMessage
}

func NewApiResponseHandler() handlex.ApiResponseHandler[handlex.Framework, RequestOption] {
	apiResponseHandler := handlex.NewApiResponseHandler[handlex.Framework, RequestOption](&handlex.ApiResponseHandlerOptions[handlex.Framework, RequestOption]{
		ResponseSuccess: func(c handlex.Framework, requestOption *RequestOption, data any) error {
			if requestOption.SuccessStatus > 0 {
				c.Status(requestOption.SuccessStatus)
			}

			streamData, ok := data.(*StreamData)
			if ok {
				if streamData.Size > 0 {
					return c.SendStream(streamData.Data, streamData.Size)
				} else {
					return c.SendStream(streamData.Data)
				}
			}
			return c.JSON(data)
		},
		ResponseError: func(c handlex.Framework, requestOption *RequestOption, err error) error {
			res := &ErrorResponse{
				Status:     false,
				StatusCode: 500,
				Code:       "E00001",
				Message:    err.Error(),
			}

			var appError *AppError
			ok := errs.As(err, &appError)
			if ok {
				res.Code = appError.ErrCode
				res.Message = appError.ErrMessage
				res.StatusCode = 400
			}
			c.Status(res.StatusCode)
			return c.JSON(res)
		},
	})
	return apiResponseHandler
}

func NewNewApiHandler() handlex.ApiHandler[handlex.Framework, RequestInfo, RequestOption] {
	requestValidator := handlex.NewRequestValidator()
	responseHandler := NewApiResponseHandler()
	return handlex.NewApiHandler[handlex.Framework, RequestInfo, RequestOption](responseHandler, &handlex.ApiHandlerOptions[handlex.Framework, RequestInfo, RequestOption]{
		OnValidate: func(c handlex.Framework, requestOption *RequestOption, data any) error {
			if requestOption.EnableValidate {
				err := requestValidator.Validate(data)
				if err != nil {
					return &AppError{ErrCode: "V0001", ErrMessage: err.Error()}
				}
				return nil
			}
			return nil
		},
		OnBefore: func(c handlex.Framework, requestOption *RequestOption) error {
			log.Println("OnBefore")
			if requestOption.EnableValidate {
				log.Println("EnableValidate")
			}
			return nil
		},
		GetRequestInfo: func(c handlex.Framework, requestOption *RequestOption) (*RequestInfo, error) {
			log.Println("GetRequestInfo")
			return &RequestInfo{
				Token: "my-token",
			}, nil
		},
		OnAfter: func(c handlex.Framework, requestOption *RequestOption) error {
			log.Println("OnAfter")
			return nil
		},
	})
}

type UploadRequest struct {
	Name  string                `form:"name"`
	File1 *multipart.FileHeader `form:"file1"`
	File2 *multipart.FileHeader `form:"file2"`
}

type SimpleRequest struct {
	Name string `json:"name"`
}

func main() {
	apiHandler := NewNewApiHandler()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return apiHandler.Do(&Fiber{Ctx: c}, nil, nil, func(ctx *handlex.Context[RequestInfo]) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/custom-status", func(c *fiber.Ctx) error {
		requestOptions := handlex.WithRequestOptions(
			SuccessStatus(201),
		)
		return apiHandler.Do(&Fiber{Ctx: c}, nil, requestOptions, func(ctx *handlex.Context[RequestInfo]) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return apiHandler.Do(&Fiber{Ctx: c}, nil, nil, func(ctx *handlex.Context[RequestInfo]) (interface{}, error) {
			return nil, &AppError{ErrCode: "0001", ErrMessage: "Error"}
		})
	})

	app.Post("/simple", func(c *fiber.Ctx) error {
		request := &SimpleRequest{}
		return apiHandler.Do(&Fiber{Ctx: c}, request, nil, func(ctx *handlex.Context[RequestInfo]) (interface{}, error) {
			fmt.Println(request.Name)
			return request.Name, nil
		})
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		request := &UploadRequest{}
		requestOptions := handlex.WithRequestOptions(
			EnableValidate(true),
		)
		return apiHandler.Do(&Fiber{Ctx: c}, request, requestOptions, func(ctx *handlex.Context[RequestInfo]) (interface{}, error) {
			fmt.Println("name:", request.Name)
			fmt.Println("file1:", request.File1.Filename)
			fmt.Println("file2:", request.File2.Filename)
			return "Success", nil
		})
	})

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}
