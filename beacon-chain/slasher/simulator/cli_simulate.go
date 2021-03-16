package simulator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/logutil"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "slasher-simulator")

// Simulate a slasher implementation at scale using different configuration
// options for observability, benchmarking, and stress-testing.
func Simulate(cliCtx *cli.Context) error {
	logrus.SetLevel(logrus.DebugLevel)
	dataDir := cliCtx.String(cmd.DataDirFlag.Name)

	logFileName := cliCtx.String(cmd.LogFileName.Name)
	if logFileName != "" {
		if err := logutil.ConfigurePersistentLogging(logFileName); err != nil {
			log.WithError(err).Error("Failed to configuring logging to disk.")
		}
	}

	// Initialize the beacon DB.
	beaconDB, err := db.NewDB(cliCtx.Context, dataDir, &kv.Config{})
	if err != nil {
		return err
	}

	// Initialize a new simulator for slasher.
	ctx, cancel := context.WithCancel(cliCtx.Context)
	defer cancel()
	sim, err := New(ctx, beaconDB)
	if err != nil {
		return err
	}

	defer func() {
		if err = sim.Stop(); err != nil {
			panic(err)
		}
	}()

	// Start the simulation.
	sim.Start()
	return nil
}