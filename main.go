package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/go-ini/ini"
	"github.com/shkh/lastfm-go/lastfm"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {

	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Println(err)
	}

	token := cfg.Section("discord").Key("token").String()
	apiKey := cfg.Section("lastfm").Key("api_key").String()
	username := cfg.Section("lastfm").Key("username").String()
	title := cfg.Section("app").Key("title").String()
	endlessMode, err := strconv.ParseBool(cfg.Section("app").Key("endless_mode").String())
	configInterval, err := cfg.Section("lastfm").Key("check_interval").Int()

	if err != nil {
		log.Println(err)
	}

	api := lastfm.New(apiKey, "")

	log.Println("Settings loaded: config.ini")

	if endlessMode {
		log.Println("Endless mode! Press `Ctrl+C` to exit")
	}
	dg, err := discordgo.New(token)
	if err != nil {
		log.Println("Discord error: ", err)
		fmt.Scanln()
		return
	}
	log.Println("Authorized to Discord")
	if err := dg.Open(); err != nil {
		log.Println("Discord error: ", err)
		fmt.Scanln()
		return
	}
	defer dg.Close()
	log.Println("Connected to Discord")

	interval := time.Duration(configInterval*1000) * time.Millisecond
	ticker := time.NewTicker(interval)

	var deathChan = make(chan os.Signal, 0)
	signal.Notify(deathChan, os.Interrupt, syscall.SIGTERM)
	go func(dg *discordgo.Session, deatchChan chan os.Signal) {
		<-deathChan
		statusData := discordgo.UpdateStatusData{Game: nil}
		if err := dg.UpdateStatusComplex(statusData); err != nil {
			log.Println("Error during deleting status:", err)
			return
		}
		log.Println("Deleting status...")
		time.Sleep(5 * time.Second)
		log.Println("Deleted status after closing")
		os.Exit(0)
	}(dg, deathChan)
	var prevTrack string
	for {
		select {
		case <-ticker.C:
			result, err := api.User.GetRecentTracks(lastfm.P{"limit": "1", "user": username})
			if err != nil {
				log.Println("LastFM error: ", err)
				if !endlessMode {
					fmt.Scanln()
					return
				}
			} else {
				if len(result.Tracks) > 0 {
					currentTrack := result.Tracks[0]
					isNowPlaying, _ := strconv.ParseBool(currentTrack.NowPlaying)
					trackName := currentTrack.Artist.Name + " - " + currentTrack.Name
					if isNowPlaying {
						statusData := discordgo.UpdateStatusData{
							Game: &discordgo.Game{
								Name:    title,
								Type:    discordgo.GameTypeListening,
								Details: currentTrack.Name,
								State:   currentTrack.Artist.Name,
							},
							AFK:    false,
							Status: "online",
						}
						if err := dg.UpdateStatusComplex(statusData); err != nil {
							log.Println("Discord error: ", err)
							if !endlessMode {
								fmt.Scanln()
								break
							}
						}
						if prevTrack != trackName {
							log.Println("Now playing: " + trackName)
							prevTrack = trackName
						}
					} else if !isNowPlaying {
						statusData := discordgo.UpdateStatusData{
							Game:   nil,
							Status: "offline",
						}
						if err := dg.UpdateStatusComplex(statusData); err != nil {
							log.Println("Discord error: ", err)
							if !endlessMode {
								fmt.Scanln()
								break
							}
						}
					}
				}
			}
		}
	}
}
