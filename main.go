package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// Function to combine chunks
func combineChunks(path string, totalChunks int, outputFilename string) error {
	outFile, err := os.OpenFile(outputFilename, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer outFile.Close()

	for i := 1; i <= totalChunks; i++ {
		chunkPath := fmt.Sprintf("%s/part%d%s", path, i, filepath.Ext(outputFilename))
		inFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func generateThumbnail(filePath string) {
	fmt.Println("Starting thumbnail generation for:", filePath)
	// Simulate thumbnail generation (replace with actual implementation)
	// Here you can call an external tool or a Go library to create the thumbnail
	fmt.Println("Thumbnail generated for:", filePath)
}

// Function to handle resumable upload
func ResumableUpload(c *gin.Context) {
	tempFolder := "./temp/"

	// Ensure the temp folder exists
	if _, err := os.Stat(tempFolder); os.IsNotExist(err) {
		fmt.Println("Creating temp folder...")
		err := os.Mkdir(tempFolder, os.ModePerm)
		if err != nil {
			c.String(http.StatusInternalServerError, "Could not create temp folder")
			return
		}
	}

	switch c.Request.Method {
	case http.MethodPost:
		file, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "Failed to get uploaded file")
			return
		}

		// Capture all parameters from the POST request
		resumableIdentifier := c.Query("resumableIdentifier")
		resumableChunkNumber := c.Query("resumableChunkNumber")
		resumableFilename := c.Query("resumableFilename")
		resumableChunkSize := c.Query("resumableChunkSize")
		resumableCurrentChunkSize := c.Query("resumableCurrentChunkSize")
		resumableTotalSize := c.Query("resumableTotalSize")
		resumableType := c.Query("resumableType")
		resumableRelativePath := c.Query("resumableRelativePath")
		resumableTotalChunks := c.Query("resumableTotalChunks")

		fmt.Printf("Received POST params: ChunkNumber=%s, ChunkSize=%s, CurrentChunkSize=%s, TotalSize=%s, Type=%s, Identifier=%s, Filename=%s, RelativePath=%s, TotalChunks=%s\n",
			resumableChunkNumber, resumableChunkSize, resumableCurrentChunkSize, resumableTotalSize, resumableType, resumableIdentifier, resumableFilename, resumableRelativePath, resumableTotalChunks)

		// Determine path and chunk file path
		fileExt := filepath.Ext(resumableFilename)
		path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier)
		relativeChunk := fmt.Sprintf("%s/part%s%s", path, resumableChunkNumber, fileExt)

		// Ensure the chunk folder exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Println("Creating directory for resumableIdentifier:", resumableIdentifier)
			err := os.Mkdir(path, os.ModePerm)
			if err != nil {
				c.String(http.StatusInternalServerError, "Could not create chunk folder")
				return
			}
		}

		// Save the uploaded chunk
		f, err := os.OpenFile(relativeChunk, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to open chunk file for writing")
			return
		}
		defer f.Close()

		src, err := file.Open()
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to open file stream")
			return
		}
		defer src.Close()

		_, err = io.Copy(f, src)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save chunk")
			return
		}

		// Check if this is the last chunk and trigger recombination if needed
		current, _ := strconv.Atoi(resumableChunkNumber)
		total, _ := strconv.Atoi(resumableTotalChunks)
		if current == total {
			fmt.Println("Combining chunks into one file")
			finalPath := fmt.Sprintf("%s/%s", tempFolder, resumableFilename)
			err = combineChunks(path, total, finalPath)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to combine chunks")
				return
			}
			fmt.Println("File recombination complete:", finalPath)

			// Start asynchronous thumbnail generation
			go generateThumbnail(finalPath)
		}

		c.String(http.StatusOK, "Chunk uploaded successfully")
	case http.MethodGet:
		// Capture all required parameters from Resumable.js
		resumableIdentifier := c.Query("resumableIdentifier")
		resumableChunkNumber := c.Query("resumableChunkNumber")
		resumableFilename := c.Query("resumableFilename")
		resumableChunkSize := c.Query("resumableChunkSize")
		resumableCurrentChunkSize := c.Query("resumableCurrentChunkSize")
		resumableTotalSize := c.Query("resumableTotalSize")
		resumableType := c.Query("resumableType")
		resumableRelativePath := c.Query("resumableRelativePath")
		resumableTotalChunks := c.Query("resumableTotalChunks")

		fmt.Printf("Received GET params: ChunkNumber=%s, ChunkSize=%s, CurrentChunkSize=%s, TotalSize=%s, Type=%s, Identifier=%s, Filename=%s, RelativePath=%s, TotalChunks=%s\n",
			resumableChunkNumber, resumableChunkSize, resumableCurrentChunkSize, resumableTotalSize, resumableType, resumableIdentifier, resumableFilename, resumableRelativePath, resumableTotalChunks)

		// Construct path for chunk storage based on identifier and chunk number
		fileExt := filepath.Ext(resumableFilename)
		path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier)
		relativeChunk := fmt.Sprintf("%s/part%s%s", path, resumableChunkNumber, fileExt)

		// Check if the chunk folder exists; create it if not
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Println("Creating directory for resumableIdentifier:", resumableIdentifier)
			err := os.Mkdir(path, os.ModePerm)
			if err != nil {
				c.String(http.StatusInternalServerError, "Could not create chunk folder")
				return
			}
		}

		// Check if the specific chunk file exists and respond accordingly
		if _, err := os.Stat(relativeChunk); os.IsNotExist(err) {
			c.String(http.StatusNotFound, "Chunk not found, ready for upload")
		} else {
			c.String(http.StatusOK, "Chunk already exists")
		}
	}

}

// Function to check upload progress
func CheckProgress(c *gin.Context) {
	tempFolder := "./temp/"
	resumableIdentifier := c.Query("resumableIdentifier")
	resumableTotalChunks := c.Query("resumableTotalChunks")

	totalChunks, err := strconv.Atoi(resumableTotalChunks)
	if err != nil || totalChunks <= 0 {
		c.String(http.StatusBadRequest, "Invalid total chunks")
		return
	}

	// Directory for chunks
	path := fmt.Sprintf("%s%s", tempFolder, resumableIdentifier)

	// Count the uploaded chunks
	uploadedChunks := 0
	for i := 1; i <= totalChunks; i++ {
		chunkPath := fmt.Sprintf("%s/part%d", path, i)
		if _, err := os.Stat(chunkPath); err == nil {
			uploadedChunks++
		}
	}

	progress := (float64(uploadedChunks) / float64(totalChunks)) * 100
	c.JSON(http.StatusOK, gin.H{"progress": fmt.Sprintf("%.2f%%", progress)})
}

func main() {
	router := gin.Default()

	// Add CORS middleware
	router.Use(cors.Default())

	// Define the upload and progress endpoints
	router.GET("/upload/progress", CheckProgress)
	router.Any("/upload", ResumableUpload)

	fmt.Println("Server started on port 8080")
	err := router.Run(":8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
