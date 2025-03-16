package api

import (
	"net/http"
	"strconv"

	"job-worker/internal/worker"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	JW worker.JobWorker
}

func NewHandler(goroutines_count, retries, timeout int) *Handler {
	h := Handler{
		JW: *worker.NewJobWorker(goroutines_count, retries, timeout),
	}
	h.JW.Start()
	return &h
}

func (h *Handler) PostJob(c echo.Context) error {
	input := struct {
		Name     *string `json:"name"`
		Priority *int    `json:"priority"`
	}{}
	err := c.Bind(&input)

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if input.Name == nil || input.Priority == nil {
		return c.String(http.StatusBadRequest, "Отсутствует хотя бы одно поле!")
	}

	id, err := h.JW.AddJob(*input.Name, *input.Priority)

	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.String(http.StatusOK, strconv.Itoa(id))
}

func (h *Handler) GetJob(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	status, ok := h.JW.StatusMap[id]
	if !ok {
		return c.String(http.StatusBadRequest, "Нет задачи с таким номером!")
	}
	return c.String(http.StatusOK, status)
}
