package main

import (
	"bufio"
	"bytes"
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

func main() {
	// Create a new Fyne app
	a := app.New()

	// Set default text style to monospace for the entire app
	a.Settings().SetTheme(&CustomTheme{a.Settings().Theme()})

	// Create a new window
	w := a.NewWindow("Fisherman")

	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(670, 280))

	// Create table headers (remove duplicate "Status" and reorder columns)
	headers := []string{"", "ID", "Image", "Created", "Ports", "Name", ""}

	// Create variables to store the current state
	var data [][]string
	var killButtons []*widget.Button
	var containers []*DockerContainer

	// Declare updateTable variable before use
	var updateTable func()

	// Create table
	var table *widget.Table
	table = widget.NewTable(
		// Function to get number of rows/cols
		func() (int, int) {
			return len(data) + 1, len(headers)
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
				label.SetText(headers[i.Col])
				label.Show()
				return
			}

			row := i.Row - 1
			if row >= len(data) {
				return
			}

			// Handle special columns
			switch i.Col {
			case 0: // Status column
				icon := "🔴"
				if strings.Contains(containers[row].Status, "Up") {
					icon = "🟢"
				}
				label.SetText(icon)
				label.Show()
			case len(headers) - 1: // Actions column
				if row < len(killButtons) {
					button.SetText("Kill")
					button.OnTapped = func() {
						c := containers[row]
						err := killDockerContainer(*c)

						if err == nil {
							updateTable()
						} else {
							a.SendNotification(&fyne.Notification{
								Title:   "Error",
								Content: fmt.Sprintf("Error killing %s container %s: %v", c.Image, c.ID, err),
							})
						}
					}
					button.Show()
				}
			default: // Regular data columns
				text := data[row][i.Col]
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

	// Adjust column widths (reduced by 50%)
	table.SetColumnWidth(0, 30)  // Status column
	table.SetColumnWidth(1, 120) // ID column
	table.SetColumnWidth(2, 75)  // Image column
	table.SetColumnWidth(3, 130) // Created column
	table.SetColumnWidth(4, 75)  // Ports column
	table.SetColumnWidth(5, 150) // Names column
	table.SetColumnWidth(6, 50)  // Actions column

	// Define updateTable function
	updateTable = func() {
		containers = getDockerContainers()
		data = make([][]string, len(containers))
		killButtons = make([]*widget.Button, len(containers))

		for i, c := range containers {
			data[i] = []string{
				"",        // Status icon column
				c.ID[:12], // Show shorter ID
				c.Image,
				c.Created,
				c.Ports,
				c.Names,
				"", // Kill button column
			}
		}

		table.Refresh()
	}

	// Initial update
	updateTable()

	// Create a ticker for live updates (every 1 second)
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			updateTable()
		}
	}()

	// Stop the ticker when the window is closed
	w.SetOnClosed(func() {
		ticker.Stop()
	})

	// Add back the Kill All button at the bottom
	killAllBtn := widget.NewButton("Kill All", func() {
		for _, c := range containers {
			if err := killDockerContainer(*c); err == nil {
				updateTable()
				a.SendNotification(&fyne.Notification{
					Title:   "Containers Killed",
					Content: "All containers have been killed",
				})
			}
		}
	})

	// Update the border container to include the Kill All button
	content := container.NewBorder(
		nil,
		killAllBtn,
		nil,
		nil,
		container.NewStack(table),
	)

	// Set the content
	w.SetContent(content)

	// Show the window
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

// getDockerContainers returns a list of Docker containers
func getDockerContainers() []*DockerContainer {
	// Use format to get exactly the fields we want in a predictable format
	// Each field is separated by a triple pipe (|||) to avoid conflicts with potential content
	format := `{{.ID}}|||{{.Image}}|||{{.Command}}|||{{.CreatedAt}}|||{{.Status}}|||{{.Ports}}|||{{.Names}}`
	cmd := exec.Command("docker", "ps", "--format", format)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	var containers []*DockerContainer
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split by our custom delimiter
		fields := strings.Split(line, "|||")
		if len(fields) != 7 {
			continue
		}

		// Parse the creation time
		createdTime, err := time.Parse("2006-01-02 15:04:05 -0700 MST", fields[3])
		if err != nil {
			// If the first format fails, try an alternative format (for RFC3339)
			createdTime, err = time.Parse(time.RFC3339Nano, fields[3])
			if err != nil {
				// If parsing fails, use the original string
				log.Printf("Error parsing time: %v", err)
				c := &DockerContainer{
					ID:      fields[0],
					Image:   fields[1],
					Command: fields[2],
					Created: fields[3], // Use original string if parsing fails
					Status:  fields[4],
					Ports:   fields[5],
					Names:   fields[6],
				}
				containers = append(containers, c)
				continue
			}
		}

		// Create a new Docker container with the exact fields
		c := &DockerContainer{
			ID:      fields[0],
			Image:   fields[1],
			Command: fields[2],
			Created: formatRelativeTime(createdTime),
			Status:  fields[4],
			Ports:   fields[5],
			Names:   fields[6],
		}

		containers = append(containers, c)
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
