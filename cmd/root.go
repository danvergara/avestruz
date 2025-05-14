/*
Copyright © 2025 Dan Vergara danvergara@nostrplebs.com
*/

package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/spf13/cobra"
)

var (
	dbURL  string
	driver string
	port   string
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
		relay.Info.Icon = "https://image.nostr.build/60a1fee58c8e5c500d32c44539ed9e2610a531ff12f627983bfae4ac323b68c9.jpg"

		db := postgresql.PostgresBackend{DatabaseURL: dbURL}

		if err := db.Init(); err != nil {
			return err
		}

		// set up the basic relay functions
		relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
		relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
		relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)

		relay.RejectEvent = append(relay.RejectEvent,
			policies.ValidateKind,
			policies.PreventLargeTags(100),
		)

		relay.RejectFilter = append(relay.RejectFilter,
			policies.NoComplexFilters,
		)

		mux := relay.Router()
		// set up other http handlers
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/html")
			_, _ = fmt.Fprintf(w, `<b>welcome</b> to my relay!`)
		})

		// start the server
		fmt.Printf("running on %s", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", port), relay); err != nil {
			return err
		}
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

	rootCmd.Flags().StringVarP(&driver, "driver", "", "", "Database Driver")
	rootCmd.Flags().StringVarP(&dbURL, "dburl", "", "", "Database DSN")
	rootCmd.Flags().StringVarP(&port, "port", "", "", "Relay Port")
}
