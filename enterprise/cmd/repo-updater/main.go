package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/inconshreveable/log15"

	ossAuthz "github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repos"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/repoupdater"
	"github.com/sourcegraph/sourcegraph/cmd/repo-updater/shared"
	frontendAuthz "github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/repo-updater/authz"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/campaigns"
	frontendDB "github.com/sourcegraph/sourcegraph/enterprise/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	ossDB "github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/ratelimit"
)

func main() {
	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	if debug {
		log.Println("enterprise edition")
	}
	shared.Main(enterpriseInit)
}

func enterpriseInit(
	db *sql.DB,
	repoStore repos.Store,
	cf *httpcli.Factory,
	server *repoupdater.Server,
) (debugDumpers []debugserver.Dumper) {
	ctx := context.Background()
	campaignsStore := campaigns.NewStore(db)

	syncRegistry := campaigns.NewSyncRegistry(ctx, campaignsStore, repoStore, cf)
	if server != nil {
		server.ChangesetSyncRegistry = syncRegistry
	}

	clock := func() time.Time {
		return time.Now().UTC().Truncate(time.Microsecond)
	}

	sourcer := repos.NewSourcer(cf)
	go campaigns.RunWorkers(ctx, campaignsStore, clock, gitserver.DefaultClient, sourcer, 5*time.Second)

	// Set up expired spec deletion
	go func() {
		for {
			if err := campaignsStore.DeleteExpiredCampaignSpecs(ctx); err != nil {
				log15.Error("DeleteExpiredCampaignSpecs", "error", err)
			}
			if err := campaignsStore.DeleteExpiredChangesetSpecs(ctx); err != nil {
				log15.Error("DeleteExpiredChangesetSpecs", "error", err)
			}

			time.Sleep(2 * time.Minute)
		}
	}()

	// TODO(jchen): This is an unfortunate compromise to not rewrite ossDB.ExternalServices for now.
	dbconn.Global = db
	permsStore := frontendDB.NewPermsStore(db, clock)
	permsSyncer := authz.NewPermsSyncer(repoStore, permsStore, clock, ratelimit.DefaultRegistry)
	go startBackgroundPermsSync(ctx, permsSyncer)
	debugDumpers = append(debugDumpers, permsSyncer)
	if server != nil {
		server.PermsSyncer = permsSyncer
	}

	return debugDumpers
}

// startBackgroundPermsSync sets up background permissions syncing.
func startBackgroundPermsSync(ctx context.Context, syncer *authz.PermsSyncer) {
	globals.WatchPermissionsUserMapping()
	go func() {
		t := time.NewTicker(5 * time.Second)
		for range t.C {
			allowAccessByDefault, authzProviders, _, _ :=
				frontendAuthz.ProvidersFromConfig(ctx, conf.Get(), ossDB.ExternalServices)
			ossAuthz.SetProviders(allowAccessByDefault, authzProviders)
		}
	}()

	go syncer.Run(ctx)
}
