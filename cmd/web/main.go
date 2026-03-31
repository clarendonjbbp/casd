package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/clarendonjbbp/casd/pkg/sorter"
)

const (
	uploadDir = "uploads"
)

//go:embed templates/*.html
var templateFS embed.FS

var pageTemplates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type resultsPageData struct {
	Logs           string
	Output         template.HTML
	Random         bool
	MinUtilization int
}

func main() {
	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handleHome)
	http.HandleFunc("/upload", handleUpload)

	log.Printf("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "home.html", nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get files
	groupsFile, err := saveUploadedFile(r, "groups")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving groups file: %v", err), http.StatusBadRequest)
		return
	}
	defer removeUploadedFile(groupsFile)

	artFile, err := saveUploadedFile(r, "art")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving art workshops file: %v", err), http.StatusBadRequest)
		return
	}
	defer removeUploadedFile(artFile)

	scienceFile, err := saveUploadedFile(r, "science")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving science workshops file: %v", err), http.StatusBadRequest)
		return
	}
	defer removeUploadedFile(scienceFile)

	// Get options
	random := r.FormValue("random") == "true"
	minUtilization := 30 // Default value
	if val := r.FormValue("min-utilization"); val != "" {
		if _, err := fmt.Sscanf(val, "%d", &minUtilization); err != nil {
			http.Error(w, fmt.Sprintf("Error parsing min-utilization: %v", err), http.StatusBadRequest)
		}
	}

	// Process files
	var buf bytes.Buffer
	log.SetOutput(&buf) // Capture log output

	groups, artWorkshops, sciWorkshops, err := sorter.ReadCSVFiles(groupsFile, artFile, scienceFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading files: %v", err), http.StatusBadRequest)
		return
	}

	if err := sorter.Schedule(groups, artWorkshops, sciWorkshops, sorter.ScheduleOptions{
		Random:                    random,
		MinUtilization:            minUtilization,
		ContinueOnParentLookupErr: true,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error scheduling workshops: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate output
	var output bytes.Buffer
	output.WriteString(`<div class="results">`)
	output.WriteString(`<details open>
		<summary><h2>Groups</h2></summary>
		<div class="section-content">`)
	sorter.PrintGroupsHTML(&output, groups)
	output.WriteString(`</div></details>`)

	output.WriteString(`<details open>
		<summary><h2>Art Workshops</h2></summary>
		<div class="section-content">`)
	sorter.PrintWorkshopsHTML(&output, artWorkshops)
	output.WriteString(`</div></details>`)

	output.WriteString(`<details open>
		<summary><h2>Science Workshops</h2></summary>
		<div class="section-content">`)
	sorter.PrintWorkshopsHTML(&output, sciWorkshops)
	output.WriteString(`</div></details>`)
	output.WriteString("</div>")

	renderTemplate(w, "results.html", resultsPageData{
		Logs:           buf.String(),
		Output:         template.HTML(output.String()),
		Random:         random,
		MinUtilization: minUtilization,
	})
}

func saveUploadedFile(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", fmt.Errorf("error getting file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("error closing uploaded file %q: %v", fieldName, closeErr)
		}
	}()

	// Create temporary file
	tempFile := filepath.Join(uploadDir, header.Filename)
	dst, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %v", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			log.Printf("error closing temp file %q: %v", tempFile, closeErr)
		}
	}()

	// Copy file contents
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("error copying file: %v", err)
	}

	return tempFile, nil
}

func removeUploadedFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("error removing uploaded file %q: %v", path, err)
	}
}

func renderTemplate(w http.ResponseWriter, name string, data any) {
	if err := pageTemplates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("unable to execute html template %q: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
