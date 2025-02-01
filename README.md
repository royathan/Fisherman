# Fisherman 🐟

Fisherman is a lightweight, user-friendly GUI application for managing Docker containers. Built with Go and the Fyne toolkit, it provides a simple interface to monitor and control your Docker containers in real-time.

## Features

- 🔄 Real-time container monitoring
- 🟢 Visual status indicators for running/stopped containers
- 🎯 One-click container management
- 📊 Detailed container information display
- 🚫 Kill individual containers or all containers at once
- 🔔 Desktop notifications for container actions

## Prerequisites

- Go 1.16 or later
- Docker installed and running
- Fyne dependencies (see Installation)

## Installation

1. Install Go if you haven't already: https://golang.org/doc/install

2. Install Fyne dependencies. On macOS:
```bash
brew install go gcc libx11-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libgl1-mesa-dev libgl1-mesa-dev xorg-dev
```

3. Clone the repository:
```bash
git clone https://github.com/yourusername/fisherman.git
cd fisherman
```

4. Install Go dependencies:
```bash
go mod tidy
```

5. Build and run:
```bash
go run main.go
```

## Usage

1. Launch the application. It will automatically detect and display all running Docker containers.

2. The main window shows a table with the following information for each container:
   - Status indicator (🟢 running, 🔴 stopped)
   - Container ID
   - Image name
   - Command
   - Creation time
   - Exposed ports
   - Container name

3. Actions:
   - Click "Kill" next to any container to stop it
   - Use "Kill All" button at the bottom to stop all containers
   - Container status updates automatically every second

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Fyne](https://fyne.io/) - Cross platform GUI toolkit
- Powered by [Docker](https://www.docker.com/) - Container platform