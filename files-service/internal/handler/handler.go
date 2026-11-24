package handler

import (
	"net/http"

	"folder.go/internal/repository"
)

type Handler struct {
	Folders    *repository.FolderRepository
	Files      *repository.FileRepository
	UsersURL   string
	StorageDir string
	HTTP       *http.Client
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.Health)

	mux.HandleFunc("POST /folders", h.CreateFolder)
	mux.HandleFunc("GET /folders", h.ListFolderContents)
	mux.HandleFunc("PATCH /folders/{id}", h.RenameFolder)

	mux.HandleFunc("POST /files", h.UploadFile)
	mux.HandleFunc("POST /files/{id}/versions", h.UploadFileVersion)
	mux.HandleFunc("PATCH /files/{id}", h.RenameFile)

	return mux
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Files service is running"))
}
