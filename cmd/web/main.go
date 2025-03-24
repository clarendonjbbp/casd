package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/clarendonjbbp/casd/pkg/sorter"
)

const (
	uploadDir      = "uploads"
	numArtSessions = 2
	numSciSessions = 2
)

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
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Workshop Scheduler</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        .options {
            margin-top: 20px;
            padding: 15px;
            background-color: #f5f5f5;
            border-radius: 5px;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        button:hover {
            background-color: #45a049;
        }
        .help-text {
            color: #666;
            font-size: 0.9em;
            margin-top: 4px;
        }
    </style>
</head>
<body>
    <h1>Workshop Scheduler</h1>
    <p>Upload your CSV files and configure scheduling options below.</p>
    
    <form action="/upload" method="post" enctype="multipart/form-data">
        <div class="form-group">
            <label for="groups">Groups CSV:</label>
            <input type="file" id="groups" name="groups" accept=".csv" required>
            <div class="help-text">CSV file containing group information</div>
        </div>
        
        <div class="form-group">
            <label for="art">Art Workshops CSV:</label>
            <input type="file" id="art" name="art" accept=".csv" required>
            <div class="help-text">CSV file containing art workshop details</div>
        </div>
        
        <div class="form-group">
            <label for="science">Science Workshops CSV:</label>
            <input type="file" id="science" name="science" accept=".csv" required>
            <div class="help-text">CSV file containing science workshop details</div>
        </div>
        
        <div class="options">
            <div class="form-group">
                <label>
                    <input type="checkbox" name="random" value="true">
                    Randomize input
                </label>
                <div class="help-text">Randomize the order of group assignments</div>
            </div>
            
            <div class="form-group">
                <label for="min-utilization">Minimum utilization (%):</label>
                <input type="number" id="min-utilization" name="min-utilization" 
                       value="30" min="0" max="100" style="width: 80px">
                <div class="help-text">Minimum percentage of workshop capacity that should be utilized</div>
            </div>
        </div>
        
        <button type="submit">Schedule Workshops</button>
    </form>
</body>
</html>
`
	t := template.Must(template.New("home").Parse(tmpl))
	if err := t.Execute(w, nil); err != nil {
		log.Printf("Unable to execute html template: %v", err)
	}
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
	defer os.Remove(groupsFile)

	artFile, err := saveUploadedFile(r, "art")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving art workshops file: %v", err), http.StatusBadRequest)
		return
	}
	defer os.Remove(artFile)

	scienceFile, err := saveUploadedFile(r, "science")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error saving science workshops file: %v", err), http.StatusBadRequest)
		return
	}
	defer os.Remove(scienceFile)

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

	groups, artWorkshops, sciWorkshops, err := readCSVFiles(groupsFile, artFile, scienceFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading files: %v", err), http.StatusBadRequest)
		return
	}

	// Book parent classes
	log.Printf("====Booking Parent Classes===\n")
	for _, group := range groups {
		for parentID := range group.ParentIDs {
			workshop, err := sorter.GetWorkshopFromID(parentID, artWorkshops, sciWorkshops)
			if err != nil {
				log.Printf("Error finding parent class for teacher=%s group=%s: %v", group.Teacher, group.Name, err)
				continue
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if !booked {
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s", parentID, group.Teacher, group.Name)
			}
		}
	}

	if random {
		shuffle(groups)
	}

	// Book art classes
	log.Printf("\n====Booking Art Classes===\n")
	bookArtClasses(groups, artWorkshops)

	// Book science classes
	log.Printf("\n====Booking Science Classes===\n")
	bookScienceClasses(groups, sciWorkshops)

	// Rebalance workshops
	log.Printf("\n====Rebalancing Workshops===\n")
	if err := rebalanceWorkshop(minUtilization, artWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance art workshops: %v", err)
	}
	if err := rebalanceWorkshop(minUtilization, sciWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance science workshops: %v", err)
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

	// Return results page
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Scheduling Results</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        pre {
            background-color: #f5f5f5;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            white-space: pre-wrap;
            font-size: 14px;
        }
        .back-link {
            margin-bottom: 20px;
        }
        .back-link a {
            color: #4CAF50;
            text-decoration: none;
        }
        .back-link a:hover {
            text-decoration: underline;
        }
        .logs {
            margin-top: 20px;
            padding: 15px;
            background-color: #fff3cd;
            border-radius: 5px;
        }
        h1, h2 {
            color: #333;
            margin-top: 1.5em;
        }
        .summary {
            margin: 20px 0;
            padding: 15px;
            background-color: #e9ecef;
            border-radius: 5px;
        }
        .results {
            margin-top: 2em;
        }
        .results details {
            margin-bottom: 2em;
            border: 1px solid #ddd;
            border-radius: 4px;
            background: white;
        }
        .results details summary {
            padding: 1em;
            cursor: pointer;
            background: #f8f9fa;
            border-bottom: 1px solid #ddd;
        }
        .results details summary:hover {
            background: #e9ecef;
        }
        .results details summary h2 {
            display: inline;
            margin: 0;
            font-size: 1.5em;
        }
        .results details summary::-webkit-details-marker {
            margin-right: 1em;
        }
        .results .section-content {
            padding: 1em;
        }
        .results pre {
            margin: 1em 0;
        }
    </style>
</head>
<body>
    <div class="back-link">
        <a href="/">← Back to Upload</a>
    </div>
    <h1>Scheduling Results</h1>
    
    {{.Output}}

    <div class="summary">
        <h3>Summary</h3>
        <p>
            Processed scheduling with:
            <ul>
                <li>Randomization: {{if .Random}}Enabled{{else}}Disabled{{end}}</li>
                <li>Minimum Utilization: {{.MinUtilization}}%</li>
            </ul>
        </p>
    </div>

    <div class="logs">
        <h2>Processing Logs</h2>
        <pre>{{.Logs}}</pre>
    </div>
</body>
</html>
`
	t := template.Must(template.New("results").Parse(tmpl))
	if err := t.Execute(w, struct {
		Logs           string
		Output         template.HTML
		Random         bool
		MinUtilization int
	}{
		Logs:           buf.String(),
		Output:         template.HTML(output.String()),
		Random:         random,
		MinUtilization: minUtilization,
	}); err != nil {
		log.Printf("Unable to execute html template: %v", err)
	}
}

func saveUploadedFile(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", fmt.Errorf("error getting file: %v", err)
	}
	defer file.Close()

	// Create temporary file
	tempFile := filepath.Join(uploadDir, header.Filename)
	dst, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %v", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("error copying file: %v", err)
	}

	return tempFile, nil
}

func readCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile string) ([]*sorter.Group, map[string]*sorter.Workshop, map[string]*sorter.Workshop, error) {
	groups, err := sorter.ReadGroups(groupsFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read groups: %v", err)
	}

	artWorkshops, err := sorter.ReadWorkshops(artWorkshopsFile, sorter.ArtWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read art workshop: %v", err)
	}

	sciWorkshops, err := sorter.ReadWorkshops(sciWorkshopsFile, sorter.SciWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read science workshop: %v", err)
	}

	return groups, artWorkshops, sciWorkshops, nil
}

func bookArtClasses(groups []*sorter.Group, artWorkshops map[string]*sorter.Workshop) {
	var needsRandomArt []*sorter.Group
	for _, group := range groups {
		sessionsToBook := numArtSessions - group.SessionsBooked(sorter.ArtWorkshop)
		if sessionsToBook < 1 {
			continue
		}
		for _, id := range group.ArtIDs {
			workshop, ok := artWorkshops[id]
			if !ok {
				log.Printf("Art workshop ID %s not found for teacher=%s group=%s", id, group.Teacher, group.Name)
				continue
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomArt = append(needsRandomArt, group)
		}
	}

	// Book random art sessions
	sortedWorkshops := sorter.SortWorkshopsByOverallUtilization(artWorkshops)
	for _, group := range needsRandomArt {
		booked := false
		for _, workshop := range sortedWorkshops {
			booked = sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				break
			}
		}
		if !booked {
			log.Printf("Could not find available art workshop for %s %s", group.Teacher, group.Name)
		}
	}
}

func bookScienceClasses(groups []*sorter.Group, sciWorkshops map[string]*sorter.Workshop) {
	var needsRandomSci []*sorter.Group
	for _, group := range groups {
		sessionsToBook := numSciSessions - group.SessionsBooked(sorter.SciWorkshop)
		if sessionsToBook < 1 {
			continue
		}
		for _, id := range group.SciIDs {
			workshop, ok := sciWorkshops[id]
			if !ok {
				log.Printf("Science workshop ID %s not found for teacher=%s group=%s", id, group.Teacher, group.Name)
				continue
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomSci = append(needsRandomSci, group)
		}
	}

	// Book random science sessions
	sortedWorkshops := sorter.SortWorkshopsByOverallUtilization(sciWorkshops)
	for _, group := range needsRandomSci {
		booked := false
		for _, workshop := range sortedWorkshops {
			booked = sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				break
			}
		}
		if !booked {
			log.Printf("Could not find available science workshop for %s %s", group.Teacher, group.Name)
		}
	}
}

func rebalanceWorkshop(minUtilization int, workshops map[string]*sorter.Workshop, groups []*sorter.Group) error {
	for maxPreferance := 1; maxPreferance < 6; maxPreferance++ {
		underutilizedWorkshops, underutilizedWorkshopSessions := sorter.GetUnderutilizedSessions(minUtilization, workshops)
		if len(underutilizedWorkshops) == 0 {
			return nil
		}
		for i := range underutilizedWorkshops {
			workshop := underutilizedWorkshops[i]
			session := underutilizedWorkshopSessions[i]
			log.Printf("Rebalancing %s at %d%% utilization for session %d", workshop.Name, workshop.Utilization(session), session)
			for _, group := range groups {
				if !workshop.WithinGradeRange(group.Grade) {
					continue
				}
				if group.IsEnrolledInWorkshop(workshop.GetID()) {
					continue
				}
				if workshop.SpotsAvailable[session] < group.NumStudents() {
					continue
				}
				oldWorkshop := group.GetWorkshop(session)
				if oldWorkshop.Kind != workshop.Kind {
					continue
				}

				if oldWorkshop.UtilizationWithoutGroup(session, group) < minUtilization {
					continue
				}
				preferance := group.HowPreferredIsBookedWorkshop(session)
				if preferance < maxPreferance {
					log.Printf("Rebalancing with group teacher=%s name=%s", group.Teacher, group.Name)
					oldWorkshop.UnbookSession(session, group)
					workshop.TakeSession(session, group)
					group.BookWorkshop(session, workshop)
					break
				}
			}
		}
	}

	return fmt.Errorf("unable to rebalance workshop")
}

func shuffle(vals []*sorter.Group) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}
