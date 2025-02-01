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
	Icon    *widget.Label
	KillBtn *widget.Button
	KillAll *widget.Button
}

func main() {
	// Create a new Fyne app
	a := app.New()

	// Create a new window
	w := a.NewWindow("Fisherman")

	// Set minimum window size
	w.Resize(fyne.NewSize(1030, 400))

	// Create table headers (remove duplicate "Status" and reorder columns)
	headers := []string{"Status", "ID", "Image", "Command", "Created", "Ports", "Names", "Actions"}

	// Create variables to store the current state
	var data [][]string
	var killButtons []*widget.Button
	var containers []*DockerContainer

	// Create table
	var table *widget.Table
	table = widget.NewTable(
		// Function to get number of rows/cols
		func() (int, int) {
			return len(data) + 1, len(headers)
		},
		// Function to create cell content
		func() fyne.CanvasObject {
			return container.NewMax(widget.NewLabel(""), widget.NewButton("Kill", nil))
		},
		// Function to update cell content
		func(i widget.TableCellID, o fyne.CanvasObject) {
			container := o.(*fyne.Container)
			label := container.Objects[0].(*widget.Label)
			button := container.Objects[1].(*widget.Button)

			// Hide both by default
			label.Show()
			button.Hide()

			if i.Col == len(headers)-1 && i.Row > 0 && i.Row-1 < len(killButtons) {
				label.Hide()
				button.Show()
				button.OnTapped = killButtons[i.Row-1].OnTapped
				return
			}

			if i.Row == 0 {
				// Header row
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.SetText(headers[i.Col])
			} else if i.Row-1 < len(data) {
				row := i.Row - 1
				if i.Col == 0 {
					// Status icon column
					icon := "🔴"
					if strings.Contains(containers[row].Status, "Up") {
						icon = "🟢"
					}
					label.SetText(icon)
				} else if i.Col < len(headers)-1 && row < len(data) {
					// Regular data columns
					text := data[row][i.Col]
					// Use fixed max lengths based on column widths
					maxLen := map[int]int{
						1: 10, // ID
						2: 15, // Image
						3: 20, // Command
						4: 10, // Created
						5: 15, // Ports
						6: 15, // Names
					}[i.Col]
					if maxLen > 0 && len(text) > maxLen {
						text = text[:maxLen] + "..."
					}
					label.SetText(text)
				}
			}
		},
	)

	// Adjust column widths
	table.SetColumnWidth(0, 60)  // Status column
	table.SetColumnWidth(1, 100) // ID column
	table.SetColumnWidth(2, 150) // Image column
	table.SetColumnWidth(3, 200) // Command column
	table.SetColumnWidth(4, 100) // Created column
	table.SetColumnWidth(5, 150) // Ports column
	table.SetColumnWidth(6, 150) // Names column
	table.SetColumnWidth(7, 80)  // Actions column

	// Function to update the table data
	updateTable := func() {
		containers = getDockerContainers()
		data = make([][]string, len(containers))
		killButtons = make([]*widget.Button, 0, len(containers))

		for i, c := range containers {
			// Create kill button
			id := c.ID
			killBtn := widget.NewButton("Kill", func() {
				killDockerContainer(id)
				a.SendNotification(&fyne.Notification{
					Title:   "Container Killed",
					Content: "Container " + id + " has been killed",
				})
			})
			killButtons = append(killButtons, killBtn)

			// Add data row
			data[i] = []string{
				"", // Status icon column
				c.ID,
				c.Image,
				c.Command,
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

	// Create a kill all button
	killAllBtn := widget.NewButton("Kill All", func() {
		for _, c := range containers {
			killDockerContainer(c.ID)
		}
		a.SendNotification(&fyne.Notification{
			Title:   "Containers Killed",
			Content: "All containers have been killed",
		})
	})

	// Create a border container with the table and kill button
	content := container.NewBorder(
		nil,
		killAllBtn,
		nil,
		nil,
		table,
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

		// Create a new icon based on the status
		if strings.Contains(c.Status, "Up") {
			c.Icon = widget.NewLabel("🟢")
		} else {
			c.Icon = widget.NewLabel("🔴")
		}

		// Create a new kill button
		id := c.ID
		c.KillBtn = widget.NewButton("Kill", func() {
			killDockerContainer(id)
			a.SendNotification(&fyne.Notification{
				Title:   "Container Killed",
				Content: "Container " + id + " has been killed",
			})
		})

		containers = append(containers, c)
	}

	return containers
}

// killDockerContainer kills a Docker container
func killDockerContainer(id string) {
	// Run the Docker kill command
	cmd := exec.Command("docker", "kill", id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	// Print the output
	fmt.Println(string(output))
}
