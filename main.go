package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var a fyne.App

// DockerContainer represents a Docker container
type DockerContainer struct {
	ID      string
	Image   string
	Command string
	Created string
	Status  string
	Ports   string
	Names   string
}

// Add this new custom theme type and implementation
type CustomTheme struct {
	fyne.Theme
}

func (t *CustomTheme) TextStyle() fyne.TextStyle {
	return fyne.TextStyle{Monospace: true}
}

// TableData holds the data and state for the container table
type TableData struct {
	headers     []string
	data        [][]string
	killButtons []*widget.Button
	containers  []*DockerContainer
	table       *widget.Table
}

// createTableData initializes a new TableData structure
func createTableData() *TableData {
	return &TableData{
		headers: []string{"", "ID", "Image", "Created", "Ports", "Name", ""},
	}
}

// createTable creates and configures the container table
func createTable(data *TableData, app fyne.App, updateFn func()) *widget.Table {
	table := widget.NewTable(
		// Function to get number of rows/cols
		func() (int, int) {
			return len(data.data) + 1, len(data.headers)
		},
		// Function to create cell content
		func() fyne.CanvasObject {
			return container.NewStack(widget.NewLabel(""), widget.NewButton("Kill", nil))
		},
		// Function to update cell content
		func(i widget.TableCellID, o fyne.CanvasObject) {
			container := o.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			button := container.Objects[1].(*widget.Button)

			// Hide both by default
			label.Hide()
			button.Hide()

			// Reset text style by default
			label.TextStyle = fyne.TextStyle{}

			if i.Row == 0 {
				// Header row
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.SetText(data.headers[i.Col])
				label.Show()
				return
			}

			row := i.Row - 1
			if row >= len(data.data) {
				return
			}

			// Handle special columns
			switch i.Col {
			case 0: // Status column
				icon := "🔴"
				if strings.Contains(data.containers[row].Status, "Up") {
					icon = "🟢"
				}
				label.SetText(icon)
				label.Show()
			case len(data.headers) - 1: // Actions column
				if row < len(data.killButtons) {
					button.SetText("Kill")
					button.OnTapped = func() {
						c := data.containers[row]
						err := killDockerContainer(*c)

						if err == nil {
							updateFn()
						} else {
							app.SendNotification(&fyne.Notification{
								Title:   "Error",
								Content: fmt.Sprintf("Error killing %s container %s: %v", c.Image, c.ID, err),
							})
						}
					}
					button.Show()
				}
			default: // Regular data columns
				text := data.data[row][i.Col]
				if maxLen := map[int]int{
					1: 12, // ID
					2: 7,  // Image
					3: 15, // Created
					4: 10, // Ports
					5: 20, // Names
				}[i.Col]; maxLen > 0 && len(text) > maxLen {
					text = text[:maxLen] + "..."
				}
				label.SetText(text)
				label.Show()
			}
		},
	)

	// Hide table dividing lines
	table.HideSeparators = true

	// Adjust column widths
	table.SetColumnWidth(0, 30)  // Status column
	table.SetColumnWidth(1, 120) // ID column
	table.SetColumnWidth(2, 75)  // Image column
	table.SetColumnWidth(3, 130) // Created column
	table.SetColumnWidth(4, 75)  // Ports column
	table.SetColumnWidth(5, 150) // Names column
	table.SetColumnWidth(6, 50)  // Actions column

	data.table = table
	return table
}

// updateTableData updates the table data with current container information
func updateTableData(data *TableData) {
	data.containers = getDockerContainers()
	data.data = make([][]string, len(data.containers))
	data.killButtons = make([]*widget.Button, len(data.containers))

	for i, c := range data.containers {
		data.data[i] = []string{
			"",        // Status icon column
			c.ID[:12], // Show shorter ID
			c.Image,
			c.Created,
			c.Ports,
			c.Names,
			"", // Kill button column
		}
	}

	if data.table != nil {
		data.table.Refresh()
	}
}

func main() {
	// Create a new Fyne app
	a := app.New()

	// Set default text style to monospace for the entire app
	a.Settings().SetTheme(&CustomTheme{a.Settings().Theme()})

	// Create a new window
	w := a.NewWindow("🎣")
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(670, 280))

	// Initialize table data
	tableData := createTableData()

	// Create update function
	updateFn := func() {
		updateTableData(tableData)
	}

	// Create table
	table := createTable(tableData, a, updateFn)

	// Initial update
	updateFn()

	// Create a ticker for live updates (every 1 second)
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			updateFn()
		}
	}()

	// Stop the ticker when the window is closed
	w.SetOnClosed(func() {
		ticker.Stop()
	})

	// Add the Kill All button at the bottom
	killAllBtn := widget.NewButton("Kill All", func() {
		for _, c := range tableData.containers {
			if err := killDockerContainer(*c); err == nil {
				updateFn()
				a.SendNotification(&fyne.Notification{
					Title:   "Containers Killed",
					Content: "All containers have been killed",
				})
			}
		}
	})

	// Create the content layout
	content := container.NewBorder(
		nil,
		killAllBtn,
		nil,
		nil,
		container.NewStack(table),
	)

	// Set the content and show the window
	w.SetContent(content)
	w.ShowAndRun()
}

// formatRelativeTime converts a timestamp to a relative time string
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 30*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// parseDockerTime attempts to parse a docker timestamp in various formats
func parseDockerTime(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339Nano,
	}

	var lastErr error
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time '%s': %v", timeStr, lastErr)
}

// getDockerContainers returns a list of Docker containers
func getDockerContainers() []*DockerContainer {
	cmd := exec.Command("docker", "ps", "--format", "{{json .}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running docker ps: %v", err)
		return nil
	}

	var containers []*DockerContainer
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var rawContainer struct {
			ID      string `json:"ID"`
			Image   string `json:"Image"`
			Command string `json:"Command"`
			Created string `json:"CreatedAt"`
			Status  string `json:"Status"`
			Ports   string `json:"Ports"`
			Names   string `json:"Names"`
		}

		if err := json.Unmarshal([]byte(line), &rawContainer); err != nil {
			log.Printf("Error parsing container JSON: %v", err)
			continue
		}

		createdTime, err := parseDockerTime(rawContainer.Created)
		created := rawContainer.Created // fallback to original string if parsing fails
		if err == nil {
			created = formatRelativeTime(createdTime)
		}

		containers = append(containers, &DockerContainer{
			ID:      rawContainer.ID,
			Image:   rawContainer.Image,
			Command: rawContainer.Command,
			Created: created,
			Status:  rawContainer.Status,
			Ports:   rawContainer.Ports,
			Names:   rawContainer.Names,
		})
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error scanning docker ps output: %v", err)
	}

	return containers
}

// killDockerContainer kills a Docker container
func killDockerContainer(container DockerContainer) error {
	cmd := exec.Command("docker", "kill", container.ID)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Error killing %s container %s: %v\n%s", container.Image, container.ID, err, output)
		return err
	}
	return nil
}
