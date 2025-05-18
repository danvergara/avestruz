/*
Copyright © 2025 Dan Vergara danvergara@nostrplebs.com
*/

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
	"github.com/jmoiron/sqlx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	dbURL              string
	driver             string
	port               string
	chacheURL          string
	ErrDuplicatedEvent = errors.New("duplicated event")
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

		opts, err := redis.ParseURL(chacheURL)
		if err != nil {
			return err
		}

		dbClient, err := sqlx.Connect("postgres", dbURL)
		if err != nil {
			return err
		}

		client := redis.NewClient(opts)

		if err := db.Init(); err != nil {
			return err
		}

		// set up the basic relay functions
		// relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
		relay.StoreEvent = append(relay.StoreEvent,
			func(ctx context.Context, event *nostr.Event) error {
				const query = `INSERT INTO event (
					id, pubkey, created_at, kind, tags, content, sig)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (id) DO NOTHING`

				var (
					tagsj, _ = json.Marshal(event.Tags)
					params   = []any{
						event.ID,
						event.PubKey,
						event.CreatedAt,
						event.Kind,
						tagsj,
						event.Content,
						event.Sig,
					}
				)

				res, err := dbClient.ExecContext(ctx, query, params...)
				if err != nil {
					return err
				}

				nr, err := res.RowsAffected()
				if err != nil {
					return err
				}

				if nr == 0 {
					return ErrDuplicatedEvent
				}

				cacheKey := fmt.Sprintf("event:%s", event.ID)
				client.HMSet(ctx, cacheKey, map[string]any{
					"pubkey":     event.PubKey,
					"created_at": event.CreatedAt,
					"kind":       event.Kind,
					"tags":       tagsj,
					"content":    event.Content,
					"sig":        event.Sig,
				})

				client.Expire(ctx, cacheKey, time.Hour)

				return nil
			},
		)

		relay.QueryEvents = append(
			relay.QueryEvents,
			func(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
				ch := make(chan *nostr.Event)
				return ch, nil
			},
		)
		// relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)

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
	rootCmd.Flags().StringVarP(&chacheURL, "cache-url", "", "", "Cache URL")
}
