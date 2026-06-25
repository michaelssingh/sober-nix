package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	searchQuery := flag.String("s", "", "Search query for anime")
	episodeNum := flag.String("e", "", "Episode number to play")
	modeFlag := flag.String("m", "dual", "Mode: sub, dub, or dual")
	qualityFlag := flag.String("q", "best", "Quality: best, worst, 1080p, 720p, etc.")
	downloadFlag := flag.Bool("d", false, "Download the episode instead of playing")
	flag.Parse()

	query := *searchQuery
	if query == "" && len(flag.Args()) > 0 {
		query = strings.Join(flag.Args(), " ")
	}

	epNo := *episodeNum
	mode := strings.ToLower(*modeFlag)

	// If query and episode are both provided, run in non-interactive direct playback mode
	if query != "" && epNo != "" {
		fmt.Printf("Non-interactive mode: Searching for %q, Episode %s...\n", query, epNo)
		shows, err := searchAnime(query, "sub")
		if err != nil {
			fmt.Printf("Error searching anime: %v\n", err)
			os.Exit(1)
		}
		if len(shows) == 0 {
			fmt.Println("No shows found.")
			os.Exit(1)
		}

		selectedShow := shows[0]
		fmt.Printf("Playing from show: %s\n", selectedShow.Name)

		if *downloadFlag {
			fmt.Println("Resolving stream for download...")
			var stream string
			if mode == "dual" || mode == "sub" {
				stream, err = resolveStreamURL(selectedShow.ID, "sub", epNo, *qualityFlag)
			} else {
				stream, err = resolveStreamURL(selectedShow.ID, "dub", epNo, *qualityFlag)
			}
			if err != nil {
				fmt.Printf("Error resolving stream: %v\n", err)
				os.Exit(1)
			}
			cmd := downloadCmd(stream, selectedShow.Name, epNo)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("Download failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Download completed successfully!")
			os.Exit(0)
		}

		var tempLua string
		var playbackLaunched bool

		if mode == "dual" {
			fmt.Println("Resolving SUB and DUB streams for dual-audio playback...")
			subStream, errSub := resolveStreamURL(selectedShow.ID, "sub", epNo, *qualityFlag)
			dubStream, errDub := resolveStreamURL(selectedShow.ID, "dub", epNo, *qualityFlag)

			if errSub != nil {
				fmt.Printf("Warning: failed to resolve SUB stream: %v. Trying DUB only.\n", errSub)
				if errDub != nil {
					fmt.Printf("Error: failed to resolve DUB stream: %v\n", errDub)
					os.Exit(1)
				}
				c, tmp, err := playSingleCmd(dubStream, selectedShow.Name, epNo)
				if err != nil {
					fmt.Printf("Error preparing playback: %v\n", err)
					os.Exit(1)
				}
				tempLua = tmp
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err == nil {
					playbackLaunched = true
				}
			} else if errDub != nil {
				fmt.Printf("Warning: failed to resolve DUB stream: %v. Playing SUB only.\n", errDub)
				c, tmp, err := playSingleCmd(subStream, selectedShow.Name, epNo)
				if err != nil {
					fmt.Printf("Error preparing playback: %v\n", err)
					os.Exit(1)
				}
				tempLua = tmp
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err == nil {
					playbackLaunched = true
				}
			} else {
				fmt.Println("Analyzing audio track count in SUB stream...")
				subTracks := countAudioStreams(subStream)
				fmt.Printf("SUB stream contains %d audio track(s).\n", subTracks)

				c, tmp, err := playDualCmd(subStream, dubStream, subTracks, selectedShow.Name, epNo)
				if err != nil {
					fmt.Printf("Error preparing playback: %v\n", err)
					os.Exit(1)
				}
				tempLua = tmp
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err == nil {
					playbackLaunched = true
				}
			}
		} else if mode == "dub" {
			dubStream, err := resolveStreamURL(selectedShow.ID, "dub", epNo, *qualityFlag)
			if err != nil {
				fmt.Printf("Error resolving DUB stream: %v\n", err)
				os.Exit(1)
			}
			c, tmp, err := playSingleCmd(dubStream, selectedShow.Name, epNo)
			if err != nil {
				fmt.Printf("Error preparing playback: %v\n", err)
				os.Exit(1)
			}
			tempLua = tmp
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err == nil {
				playbackLaunched = true
			}
		} else {
			subStream, err := resolveStreamURL(selectedShow.ID, "sub", epNo, *qualityFlag)
			if err != nil {
				fmt.Printf("Error resolving SUB stream: %v\n", err)
				os.Exit(1)
			}
			c, tmp, err := playSingleCmd(subStream, selectedShow.Name, epNo)
			if err != nil {
				fmt.Printf("Error preparing playback: %v\n", err)
				os.Exit(1)
			}
			tempLua = tmp
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err == nil {
				playbackLaunched = true
			}
		}

		if tempLua != "" {
			_ = os.Remove(tempLua)
		}
		if playbackLaunched {
			_ = recordWatch(selectedShow.ID, selectedShow.Name, epNo)
		}
		os.Exit(0)
	}

	// Interactive TUI mode
	p := tea.NewProgram(initialModel(query, mode, *qualityFlag, *downloadFlag), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func debugLog(format string, args ...any) {
	if os.Getenv("CLARE_DEBUG") == "" {
		return
	}
	dir := os.Getenv("CLARE_STATE_DIR")
	if dir == "" {
		stateHome := os.Getenv("XDG_STATE_HOME")
		if stateHome == "" {
			home, _ := os.UserHomeDir()
			stateHome = filepath.Join(home, ".local", "state")
		}
		dir = filepath.Join(stateHome, "clare")
	}
	_ = os.MkdirAll(dir, 0755)
	logFile := filepath.Join(dir, "debug.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] "+format+"\n", append([]any{time.Now().Format("15:04:05")}, args...)...)
}
