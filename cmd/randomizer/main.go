package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

var teacherFirstNames = []string{
	"Emma", "Liam", "Olivia", "Noah", "Ava", "Oliver", "Isabella", "William", "Sophia", "James",
	"Charlotte", "Benjamin", "Amelia", "Lucas", "Mia", "Henry", "Harper", "Theodore", "Evelyn", "Jack",
	"Elizabeth", "Alexander", "Sofia", "Sebastian", "Avery", "Michael", "Ella", "Daniel", "Scarlett", "Owen",
	"Luna", "Samuel", "Victoria", "Joseph", "Grace", "David", "Chloe", "Carter", "Penelope", "John",
	"Layla", "Gabriel", "Riley", "Luke", "Zoey", "Anthony", "Nora", "Isaac", "Lily", "Dylan",
}

var studentFirstNames = []string{
	"Ethan", "Sophia", "Mason", "Ava", "Lucas", "Isabella", "Jackson", "Mia", "Aiden", "Charlotte",
	"Elijah", "Amelia", "Oliver", "Harper", "Jacob", "Evelyn", "Lucas", "Abigail", "Michael", "Emily",
	"Alexander", "Elizabeth", "James", "Sofia", "Benjamin", "Avery", "Matthew", "Ella", "William", "Scarlett",
	"Daniel", "Victoria", "Henry", "Madison", "Joseph", "Luna", "Samuel", "Grace", "Sebastian", "Chloe",
	"David", "Penelope", "Carter", "Layla", "Owen", "Riley", "Wyatt", "Zoey", "Luke", "Nora",
	"Gabriel", "Hannah", "Dylan", "Lily", "Nathan", "Eleanor", "Isaac", "Hazel", "Caleb", "Violet",
	"Ryan", "Aurora", "Adrian", "Nova", "Ian", "Emilia", "Adam", "Naomi", "Eli", "Maya",
	"Jonathan", "Eva", "Theodore", "Alice", "Nicholas", "Sarah", "Aaron", "Ariana", "Connor", "Clara",
	"Cameron", "Lucy", "Thomas", "Anna", "Robert", "Audrey", "Josiah", "Bella", "Austin", "Skylar",
	"Jeremiah", "Claire", "Xavier", "Paisley", "Kevin", "Julia", "Dominic", "Ruby", "Kyle", "Rose",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
	"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
	"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
	"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell", "Carter", "Roberts",
	"Chen", "Zhang", "Kim", "Singh", "Patel", "Kumar", "Shah", "Li", "Wang", "Yang",
	"Wu", "Liu", "Huang", "Park", "Choi", "Jung", "Cho", "Kang", "Gupta", "Sharma",
	"Murphy", "O'Connor", "Walsh", "Ryan", "O'Brien", "McCarthy", "Sullivan", "O'Neill", "Byrne", "Kelly",
	"Anderson", "Eriksson", "Larsson", "Nilsson", "Karlsson", "Olsson", "Svensson", "Gustafsson", "Berg", "Holm",
	"Cohen", "Levy", "Goldberg", "Friedman", "Katz", "Stern", "Rosen", "Klein", "Schwartz", "Weiss",
}

func main() {
	// Parse command line arguments
	inputFile := flag.String("input", "groups.csv", "Input CSV file to process")
	outputFile := flag.String("output", "", "Output CSV file (default: input_randomized.csv)")
	flag.Parse()

	// If no output file specified, create one based on input filename
	if *outputFile == "" {
		ext := filepath.Ext(*inputFile)
		base := strings.TrimSuffix(*inputFile, ext)
		*outputFile = base + "_randomized" + ext
	}

	if err := randomizeNames(*inputFile, *outputFile); err != nil {
		log.Fatal(err)
	}
}

type nameMapping struct {
	firstName string
	lastName  string
}

func randomizeNames(inputFile, outputFile string) error {
	// Read the input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("error closing input file: %v", closeErr)
		}
	}()

	// Create a CSV reader
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %v", err)
	}

	// Create maps to maintain consistent name replacements
	teacherMap := make(map[string]string)
	studentMap := make(map[string]nameMapping)

	// Process each row
	for i := 1; i < len(records); i++ { // Skip header row
		// Handle teacher name (column 0)
		originalTeacherName := strings.TrimSpace(records[i][0])
		if originalTeacherName != "" {
			if replacement, exists := teacherMap[originalTeacherName]; exists {
				records[i][0] = replacement
			} else {
				newName := teacherFirstNames[rand.Intn(len(teacherFirstNames))]
				teacherMap[originalTeacherName] = newName
				records[i][0] = newName
			}
		}

		// Handle student names (column 4)
		if len(records[i]) > 4 {
			studentNamesStr := records[i][4]
			if studentNamesStr != "" {
				studentNames := splitStudentNames(studentNamesStr)
				var newStudentNames []string

				for _, studentName := range studentNames {
					studentName = strings.TrimSpace(studentName)
					if studentName == "" {
						continue
					}

					if replacement, exists := studentMap[studentName]; exists {
						newStudentNames = append(newStudentNames, replacement.firstName+" "+replacement.lastName)
					} else {
						newFirstName := studentFirstNames[rand.Intn(len(studentFirstNames))]
						newLastName := lastNames[rand.Intn(len(lastNames))]
						studentMap[studentName] = nameMapping{firstName: newFirstName, lastName: newLastName}
						newStudentNames = append(newStudentNames, newFirstName+" "+newLastName)
					}
				}

				// Join the names back together
				records[i][4] = strings.Join(newStudentNames, ", ")
			}
		}
	}

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			log.Printf("error closing output file: %v", closeErr)
		}
	}()

	// Write the modified records
	writer := csv.NewWriter(outFile)
	err = writer.WriteAll(records)
	if err != nil {
		return fmt.Errorf("error writing CSV: %v", err)
	}

	fmt.Printf("Successfully created %s with randomized names\n\n", outputFile)
	fmt.Println("Original to New Teacher Name Mapping:")
	for orig, new := range teacherMap {
		fmt.Printf("%s -> %s\n", orig, new)
	}

	fmt.Println("\nOriginal to New Student Name Mapping:")
	for orig, new := range studentMap {
		fmt.Printf("%s -> %s %s\n", orig, new.firstName, new.lastName)
	}

	return nil
}

func splitStudentNames(studentNames string) []string {
	normalized := strings.ReplaceAll(studentNames, "\t", " ")

	var rawNames []string
	switch {
	case strings.Contains(normalized, ";"):
		rawNames = strings.Split(normalized, ";")
	case strings.Contains(normalized, "\n"):
		rawNames = strings.Split(normalized, "\n")
	default:
		rawNames = strings.Split(normalized, ",")
	}

	names := make([]string, 0, len(rawNames))
	for _, name := range rawNames {
		name = strings.TrimSpace(name)
		name = strings.TrimPrefix(name, "and ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	return names
}
