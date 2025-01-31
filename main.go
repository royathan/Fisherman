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
	w.Resize(fyne.NewSize(800, 400))

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
						1: 10,  // ID
						2: 15,  // Image
						3: 20,  // Command
						4: 10,  // Created
						5: 15,  // Ports
						6: 15,  // Names
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

// getDockerContainers returns a list of Docker containers
func getDockerContainers() []*DockerContainer {
	// Run the Docker ps command with full container IDs
	cmd := exec.Command("docker", "ps", "--no-trunc")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	// Parse the output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var containers []*DockerContainer
	scanner.Scan() // Skip the header

	for scanner.Scan() {
		line := scanner.Text()

		// Find the command portion (everything between quotes)
		commandStart := strings.Index(line, "\"")
		commandEnd := strings.LastIndex(line, "\"")
		command := ""
		if commandStart != -1 && commandEnd != -1 && commandEnd > commandStart {
			command = line[commandStart+1 : commandEnd]
			// Remove the command portion from the line for proper field splitting
			line = line[:commandStart] + line[commandEnd+1:]
		}

		// Split remaining fields
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Find status and ports
		var status, ports string
		for i := 4; i < len(fields)-1; i++ {
			if strings.Contains(fields[i], "Up") || strings.Contains(fields[i], "Exited") {
				status = strings.Join(fields[i:i+2], " ")
				if i+2 < len(fields)-1 {
					ports = strings.Join(fields[i+2:len(fields)-1], " ")
				}
				break
			}
		}

		// Create a new Docker container
		c := &DockerContainer{
			ID:      fields[0],
			Image:   fields[1],
			Command: command,
			Created: fields[2] + " " + fields[3],
			Status:  status,
			Ports:   ports,
			Names:   fields[len(fields)-1],
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
