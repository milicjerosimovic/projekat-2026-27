package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"folder.go/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateFolderRequest struct {
	Name     string `json:"name"`
	OwnerID  string `json:"ownerId"`
	ParentID string `json:"parentId"`
}

func (h *Handler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var request CreateFolderRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	request.Name = strings.TrimSpace(request.Name)

	if request.Name == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}

	ownerID, err := primitive.ObjectIDFromHex(request.OwnerID)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	var parentID *primitive.ObjectID

	if request.ParentID != "" {
		id, err := primitive.ObjectIDFromHex(request.ParentID)
		if err != nil {
			http.Error(w, "Invalid parent ID", http.StatusBadRequest)
			return
		}

		_, err = h.Folders.GetFolderByID(id, ownerID)
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Parent folder not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		parentID = &id
	}

	folder := model.Folder{
		Name:     request.Name,
		OwnerID:  ownerID,
		ParentID: parentID,
	}

	err = h.Folders.CreateFolder(&folder)
	if err != nil {
		http.Error(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func (h *Handler) ListFolderContents(w http.ResponseWriter, r *http.Request) {
	ownerID, err := primitive.ObjectIDFromHex(r.URL.Query().Get("ownerId"))
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	parentIDString := r.URL.Query().Get("parentId")
	var parentID *primitive.ObjectID

	if parentIDString != "" {
		id, err := primitive.ObjectIDFromHex(parentIDString)
		if err != nil {
			http.Error(w, "Invalid parent ID", http.StatusBadRequest)
			return
		}

		_, err = h.Folders.GetFolderByID(id, ownerID)
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Folder not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		parentID = &id
	}

	folders, err := h.Folders.GetFoldersByParentID(ownerID, parentID)
	if err != nil {
		http.Error(w, "Failed to load folders", http.StatusInternalServerError)
		return
	}

	files := make([]model.File, 0)

	if parentID != nil {
		files, err = h.Files.GetFilesByParentID(ownerID, *parentID)
		if err != nil {
			http.Error(w, "Failed to load files", http.StatusInternalServerError)
			return
		}
	}

	response := struct {
		Folders []model.Folder `json:"folders"`
		Files   []model.File   `json:"files"`
	}{
		Folders: folders,
		Files:   files,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type RenameFolderRequest struct {
	OwnerID string `json:"ownerId"`
	Name    string `json:"name"`
}

func (h *Handler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	folderID, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	var request RenameFolderRequest

	err = json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ownerID, err := primitive.ObjectIDFromHex(request.OwnerID)
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}

	err = h.Folders.RenameFolder(folderID, ownerID, request.Name)
	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to rename folder", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Folder renamed successfully"))
}
