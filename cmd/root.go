/*
Copyright © 2025 Dan Vergara danvergara@nostrplebs.com
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "avestruz",
	Short: "Cloud Native and Self-Hosting friendly Nostr Relay",
	Long:  `Cloud Native and Self-Hosting friendly Nostr Relay`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// create the relay instance

		relay := khatru.NewRelay()

		// set up some basic properties (will be returned on the NIP-11 endpoint)
		relay.Info.Name = "avestruz relay"
		relay.Info.PubKey = "0d70ebe457331dc3a47b8d6cbad9da410a88b874414271a2dece4c35b371f044"
		relay.Info.Description = "Official avestruz relay"
		relay.Info.Icon = "https://www.google.com/imgres?q=purple%20ostrich&imgurl=https%3A%2F%2Fwww.redbrokoly.com%2F172889-large_default%2Fpurple-ostrich-mascot-costume-character-dressed-with-a-turtleneck-and-lapel-pins.jpg&imgrefurl=https%3A%2F%2Fwww.redbrokoly.com%2Fen%2Fostrich-mascots%2F160534-purple-ostrich-mascot-costume-character-dressed-with-a-turtleneck-and-lapel-pins.html&docid=RKKQiCbAu8hIIM&tbnid=elw9VcSeHLyeKM&vet=12ahUKEwjyoMbrlpONAxWj38kDHS8YMX0QM3oECG4QAA..i&w=800&h=800&hcb=2&itg=1&ved=2ahUKEwjyoMbrlpONAxWj38kDHS8YMX0QM3oECG4QAA"

		// you must bring your own storage scheme -- if you want to have any
		store := make(map[string]*nostr.Event, 120)

		// set up the basic relay functions
		relay.StoreEvent = append(relay.StoreEvent,
			func(ctx context.Context, event *nostr.Event) error {
				store[event.ID] = event
				return nil
			},
		)
		relay.QueryEvents = append(relay.QueryEvents,
			func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
				ch := make(chan *nostr.Event)
				go func() {
					for _, evt := range store {
						if filter.Matches(evt) {
							ch <- evt
						}
					}
					close(ch)
				}()
				return ch, nil
			},
		)
		relay.DeleteEvent = append(relay.DeleteEvent,
			func(ctx context.Context, event *nostr.Event) error {
				delete(store, event.ID)
				return nil
			},
		)

		// there are many other configurable things you can set
		relay.RejectEvent = append(relay.RejectEvent,
			// built-in policies
			policies.ValidateKind,
			// define your own policies
			policies.PreventLargeTags(100),
			func(ctx context.Context, event *nostr.Event) (reject bool, msg string) {
				if event.PubKey == "fa984bd7dbb282f07e16e7ae87b26a2a7b9b90b7246a44771f0cf5ae58018f52" {
					return true, "we don't allow this person to write here"
				}
				return false, "" // anyone else can
			},
		)

		// you can request auth by rejecting an event or a request with the prefix "auth-required: "
		relay.RejectFilter = append(relay.RejectFilter,
			// built-in policies
			policies.NoComplexFilters,

			// define your own policies
			func(ctx context.Context, filter nostr.Filter) (reject bool, msg string) {
				if pubkey := khatru.GetAuthed(ctx); pubkey != "" {
					log.Printf("request from %s\n", pubkey)
					return false, ""
				}
				return true, "auth-required: only authenticated users can read from this relay"
				// (this will cause an AUTH message to be sent and then a CLOSED message such that clients can
				//  authenticate and then request again)
			},
		)
		// check the docs for more goodies!

		mux := relay.Router()
		// set up other http handlers
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/html")
			fmt.Fprintf(w, `<b>welcome</b> to my relay!`)
		})

		// start the server
		fmt.Println("running on :3334")
		http.ListenAndServe(":3334", relay)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.avestruz.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
