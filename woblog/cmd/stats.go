package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type server struct {
	ID    string
	Name  string
	Count int
}

func init() {
	rootCmd.AddCommand(stats)
}

var stats = &cobra.Command{
	Use:   "stats",
	Short: "Parse an access log and display relevant stats",
	Run:   runStats,
}

// TODO open file over ssh?
func runStats(cmd *cobra.Command, args []string) {
	fp, err := os.Open(logFile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer fp.Close()

	servers := make(map[string]*server)
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "\t")
		if line[1] == "" {
			continue
		}
		s, ok := servers[line[1]]
		if !ok {
			guild, err := session().Guild(line[1])
			if err != nil {
				log.Println(err)
				continue
			}
			s = &server{
				ID:    line[1],
				Name:  guild.Name,
				Count: 0,
			}
			servers[line[1]] = s
		}
		s.Count++
	}

	for _, v := range servers {
		fmt.Printf("%v: %s\n", v.Count, v.Name)
	}
}
