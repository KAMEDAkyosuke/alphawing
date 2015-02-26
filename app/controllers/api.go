package controllers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/kayac/alphawing/app/models"

	"github.com/revel/revel"
)

type JsonResponse struct {
	Status  int      `json:"status"`
	Message []string `json:"message"`
}

type JsonResponseUploadBundle struct {
	*JsonResponse
	Content *models.BundleJsonResponse `json:"content"`
}

type ApiController struct {
	AlphaWingController
}

func (c ApiController) NewJsonResponse(stat int, mes []string) *JsonResponse {
	return &JsonResponse{
		Status:  stat,
		Message: mes,
	}
}

func (c ApiController) NewJsonResponseUploadBundle(stat int, mes []string, content *models.BundleJsonResponse) *JsonResponseUploadBundle {
	return &JsonResponseUploadBundle{
		c.NewJsonResponse(stat, mes),
		content,
	}
}

func (c ApiController) GetDocument() revel.Result {
	return c.Render()
}

func (c ApiController) PostUploadBundle(token string, description string, file *os.File) revel.Result {
	app, err := models.GetAppByApiToken(Dbm, token)
	if err != nil {
		c.Response.Status = http.StatusUnauthorized
		return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, []string{"Token is invalid."}, nil))
	}

	var filename string
	if _, ok := c.Params.Files["file"]; ok {
		filename = c.Params.Files["file"][0].Filename
	}
	extStr := filepath.Ext(filename)
	ext := models.BundleFileExtension(extStr)
	isValidExt := ext.IsValid()

	c.Validation.Required(file != nil).Message("File is required.")
	c.Validation.Required(isValidExt).Message("File extension is not valid.")
	if c.Validation.HasErrors() {
		var errors []string
		for _, err := range c.Validation.Errors {
			errors = append(errors, err.String())
		}
		c.Response.Status = http.StatusBadRequest
		return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, errors, nil))
	}

	bundle := &models.Bundle{
		PlatformType: ext.PlatformType(),
		Description:  description,
		File:         file,
	}

	if err := app.CreateBundle(Dbm, c.GoogleService, bundle); err != nil {
		if bperr, ok := err.(*models.BundleParseError); ok {
			c.Response.Status = http.StatusInternalServerError
			return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, []string{bperr.Error()}, nil))
		}
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, []string{err.Error()}, nil))
	}

	content, err := bundle.JsonResponse(&c)
	if err != nil {
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, []string{err.Error()}, nil))
	}

	c.Response.Status = http.StatusOK
	return c.RenderJson(c.NewJsonResponseUploadBundle(c.Response.Status, []string{"Bundle is created!"}, content))
}
