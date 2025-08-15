package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/josephburnett/jd/v2"
	"gopkg.in/yaml.v3"
)

var fileName = flag.String("f", "", "Path to input file (leave empty to read from stdin)")
var color = flag.Bool("c", false, "Enable color output")

// TerraformPlan represents the structure of the Terraform plan JSON file
type TerraformPlan struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

// ResourceChange represents a change to a resource in the Terraform plan
type ResourceChange struct {
	Address string `json:"address"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Change  Change `json:"change"`
}

// Change represents the change details for a resource
type Change struct {
	Actions []string       `json:"actions"`
	Before  map[string]any `json:"before"`
	After   map[string]any `json:"after"`
}

// ProcessedResource represents a resource with processed before/after states
type ProcessedResource struct {
	Address   string
	Operation string
	Before    map[string]any
	After     map[string]any
	Diff      string
}

// Main function to parse the Terraform plan and output differences
func main() {
	flag.Parse()

	var scanner *bufio.Scanner
	if fileName != nil && *fileName != "" {
		planFile := *fileName

		if _, err := os.Stat(planFile); os.IsNotExist(err) {
			fmt.Printf("Error: Plan file not found: %s\n", planFile)
			os.Exit(1)
		}

		planF, err := os.Open(planFile)
		if err != nil {
			fmt.Printf("Error: Failed to open plan file: %s: %s\n", planFile, err)
			os.Exit(1)
		}
		defer planF.Close()
		scanner = bufio.NewScanner(planF)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	input := &strings.Builder{}
	for scanner.Scan() {
		input.WriteString(scanner.Text())
	}

	var tfPlan TerraformPlan
	err := json.Unmarshal([]byte(input.String()), &tfPlan)
	if err != nil {
		fmt.Printf("Error: Failed to parse input: %s\n", err)
		os.Exit(1)
	}

	var processedResources []ProcessedResource
	for _, rc := range tfPlan.ResourceChanges {
		processedResource, ok := processResourceChange(rc)
		if !ok {
			continue
		} else {
			processedResources = append(processedResources, processedResource)
		}
	}

	if len(processedResources) == 0 {
		fmt.Println("No changes detected")
	} else {
		fmt.Printf("%d changes detected:\n\n", len(processedResources))
		for i, pr := range processedResources {
			fmt.Println("#########################################")
			fmt.Printf("%d. Change (%s)\n", i+1, pr.Address)
			fmt.Printf("Address: %s\n", pr.Address)
			fmt.Printf("Operation: %s\n", pr.Operation)
			fmt.Println("Diff:")
			fmt.Println()
			fmt.Println(pr.Diff)
			fmt.Println("- - - - - - - - - - - - - - - - - - - - -")
		}
	}
	fmt.Println("Done")
}

func processResourceChange(resourceChange ResourceChange) (ProcessedResource, bool) {
	if len(resourceChange.Change.Actions) == 0 || resourceChange.Change.Actions[0] == "no-op" {
		return ProcessedResource{}, false
	}

	before := parseAsJSON(resourceChange.Change.Before)
	after := parseAsJSON(resourceChange.Change.After)

	beforeB, err := json.Marshal(before)
	if err != nil {
		log.Printf("Error marshalling before state: %s\n", err)
		return ProcessedResource{}, false
	}

	afterB, err := json.Marshal(after)
	if err != nil {
		log.Printf("Error marshalling after state: %s\n", err)
		return ProcessedResource{}, false
	}

	beforeNode, err := jd.ReadJsonString(string(beforeB))
	if err != nil {
		log.Printf("Error parsing before state: %s\n", err)
		return ProcessedResource{}, false
	}

	afterNode, err := jd.ReadJsonString(string(afterB))
	if err != nil {
		log.Printf("Error parsing after state: %s\n", err)
		return ProcessedResource{}, false
	}

	var diff string
	if color != nil && *color {
		diff = beforeNode.Diff(afterNode).Render(jd.COLOR)
	} else {
		diff = beforeNode.Diff(afterNode).Render()
	}

	return ProcessedResource{
		Address:   resourceChange.Address,
		Operation: resourceChange.Change.Actions[0],
		Before:    before.(map[string]any),
		After:     after.(map[string]any),
		Diff:      diff,
	}, true
}

var spaces = 0
var turnOnIndentLogging = false

func parseAsJSON(value any) any {
	for i := 0; i < spaces*2; i++ {
		if turnOnIndentLogging {
			fmt.Print(" ")
		}
	}
	if turnOnIndentLogging {
		fmt.Printf("%v\n", value)
	}
	defer func() {
		spaces--
		for i := 0; i < spaces*2; i++ {
			if turnOnIndentLogging {
				fmt.Print(" ")
			}
		}
		if turnOnIndentLogging {
			fmt.Print("done with this inline\n")
		}
	}()

	spaces++
	switch val := value.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = parseAsJSON(v)
		}
		return result
	case string:
		if json.Valid([]byte(val)) {
			var parsedStrAsJSON any
			if err := json.Unmarshal([]byte(val), &parsedStrAsJSON); err != nil {
				log.Printf("Error unmarshalling JSON: %s\n", val)
				return val
			}
			return parseAsJSON(parsedStrAsJSON)
		} else {
			var parsedStrAsYAML map[string]any
			if err := yaml.Unmarshal([]byte(val), &parsedStrAsYAML); err != nil {
				return val
			} else {
				return parseAsJSON(parsedStrAsYAML)
			}
		}
	case []interface{}:
		processedSlice := make([]any, 0, len(val))
		for _, s := range val {
			processedSlice = append(processedSlice, parseAsJSON(s))
		}
		return processedSlice

	default:
		return val
	}
}
