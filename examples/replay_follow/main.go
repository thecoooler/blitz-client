// Library-only live replay consumer. Ctrl+C stops the example.
package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"time"

	blitzclient "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/replay"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: replay_follow <data.replay>")
	}
	follower := replay.NewFollower(os.Args[1])
	out := bufio.NewWriter(os.Stdout)
	for {
		_, err := follower.Poll(func(event replay.FollowEvent) error {
			line, err := json.Marshal(&event)
			if err != nil {
				return blitzclient.NewError("JSON_SERIALIZE_FAILED", err.Error())
			}
			line = append(line, '\n')
			if _, err := out.Write(line); err != nil {
				return blitzclient.NewError("OUTPUT_FAILED", err.Error())
			}
			if err := out.Flush(); err != nil {
				return blitzclient.NewError("OUTPUT_FAILED", err.Error())
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}
