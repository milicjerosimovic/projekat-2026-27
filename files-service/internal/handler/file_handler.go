package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"folder.go/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RenameFileRequest struct {
	OwnerID string `json:"ownerId"`
	Name    string `json:"name"`
}

func (h *Handler) CheckQuota(
	w http.ResponseWriter,
	r *http.Request,
	ownerID primitive.ObjectID,
	additionalSize int64,
) bool {

	user, err := h.GetUserInfo(
		r.Context(),
		r.Header.Get("Authorization"),
	)

	if err != nil {
		http.Error(w, "User verification failed", http.StatusUnauthorized)
		return false
	}

	if user.ID != ownerID.Hex() {
		http.Error(w, "Access denied", http.StatusForbidden)
		return false
	}

	usedStorage, err := h.Files.GetTotalStorageUsed(ownerID)

	if err != nil {
		http.Error(w, "Failed to calculate storage", http.StatusInternalServerError)
		return false
	}

	if additionalSize > 0 &&
		additionalSize > user.QuotaBytes-usedStorage {
		http.Error(w, "Storage quota exceeded", http.StatusForbidden)
		return false
	}

	return true
}

func (h *Handler) RenameFile(w http.ResponseWriter, r *http.Request) {
	fileID, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	var request RenameFileRequest

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
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	err = h.Files.RenameFile(fileID, ownerID, request.Name)

	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Failed to rename file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File renamed successfully"))
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Invalid upload request", http.StatusBadRequest)
		return
	}

	defer r.MultipartForm.RemoveAll()

	ownerID, err := primitive.ObjectIDFromHex(r.FormValue("ownerId"))
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	parentID, err := primitive.ObjectIDFromHex(r.FormValue("parentId"))
	if err != nil {
		http.Error(w, "Invalid parent ID", http.StatusBadRequest)
		return
	}

	_, err = h.Folders.GetFolderByID(parentID, ownerID)

	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}

	defer uploadedFile.Close()

	if !h.CheckQuota(w, r, ownerID, fileHeader.Size) {
		return
	}

	fileID := primitive.NewObjectID()

	err = os.MkdirAll(h.StorageDir, 0755)
	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	contentRef := filepath.Join(h.StorageDir, fileID.Hex())

	destination, err := os.Create(contentRef)
	if err != nil {
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	size, copyErr := io.Copy(destination, uploadedFile)
	closeErr := destination.Close()

	if copyErr != nil || closeErr != nil {
		os.Remove(contentRef)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	file := model.File{
		ID:         fileID,
		OwnerID:    ownerID,
		ParentID:   parentID,
		Name:       filepath.Base(fileHeader.Filename),
		Size:       size,
		Type:       fileHeader.Header.Get("Content-Type"),
		ContentRef: contentRef,
	}

	err = h.Files.CreateFile(&file)
	if err != nil {
		os.Remove(contentRef)
		http.Error(w, "Failed to save file metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(file)
}

func (h *Handler) UploadFileVersion(w http.ResponseWriter, r *http.Request) {
	fileID, err := primitive.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Invalid upload request", http.StatusBadRequest)
		return
	}

	defer r.MultipartForm.RemoveAll()

	ownerID, err := primitive.ObjectIDFromHex(r.FormValue("ownerId"))
	if err != nil {
		http.Error(w, "Invalid owner ID", http.StatusBadRequest)
		return
	}

	existingFile, err := h.Files.GetFileByID(fileID, ownerID)

	if errors.Is(err, mongo.ErrNoDocuments) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	uploadedFile, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is required", http.StatusBadRequest)
		return
	}

	defer uploadedFile.Close()

	additionalSize := fileHeader.Size - existingFile.Size

	if !h.CheckQuota(w, r, ownerID, additionalSize) {
		return
	}

	err = os.MkdirAll(h.StorageDir, 0755)
	if err != nil {
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	newContentRef := filepath.Join(
		h.StorageDir,
		primitive.NewObjectID().Hex(),
	)

	destination, err := os.Create(newContentRef)
	if err != nil {
		http.Error(w, "Cannot save file", http.StatusInternalServerError)
		return
	}

	size, copyErr := io.Copy(destination, uploadedFile)
	closeErr := destination.Close()

	if copyErr != nil || closeErr != nil {
		os.Remove(newContentRef)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	err = h.Files.UpdateFileVersion(
		fileID,
		ownerID,
		size,
		fileHeader.Header.Get("Content-Type"),
		newContentRef,
	)

	if err != nil {
		os.Remove(newContentRef)
		http.Error(w, "Failed to update file version", http.StatusInternalServerError)
		return
	}

	if existingFile.ContentRef != "" {
		os.Remove(existingFile.ContentRef)
	}

	updatedFile, err := h.Files.GetFileByID(fileID, ownerID)

	if err != nil {
		http.Error(w, "Failed to load updated file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedFile)
}
