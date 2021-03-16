package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_processQueuedBlocks_DetectsDoubleProposals(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			Database:              beaconDB,
			ProposerSlashingsFeed: new(event.Feed),
		},
		params:    DefaultParams(),
		blksQueue: newBlocksQueue(),
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedBlocks(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()
	s.blksQueue.extend([]*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, 4, 1, []byte{1}),
		createProposalWrapper(t, 4, 1, []byte{1}),
		createProposalWrapper(t, 4, 1, []byte{1}),
		createProposalWrapper(t, 4, 1, []byte{2}),
	})
	currentSlot := types.Slot(4)
	currentSlotChan <- currentSlot
	cancel()
	<-exitChan
	require.LogsContain(t, hook, "Proposer double proposal slashing")
}

func Test_processQueuedBlocks_NotSlashable(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
		params:    DefaultParams(),
		blksQueue: newBlocksQueue(),
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedBlocks(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()
	s.blksQueue.extend([]*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, 4, 1, []byte{1}),
		createProposalWrapper(t, 4, 1, []byte{1}),
	})
	currentSlot := types.Slot(4)
	currentSlotChan <- currentSlot
	cancel()
	<-exitChan
	require.LogsDoNotContain(t, hook, "Proposer double proposal slashing")
}

func createProposalWrapper(t *testing.T, slot types.Slot, proposerIndex types.ValidatorIndex, signingRoot []byte) *slashertypes.SignedBlockHeaderWrapper {
	header := &ethpb.BeaconBlockHeader{
		Slot:          slot,
		ProposerIndex: proposerIndex,
		ParentRoot:    params.BeaconConfig().ZeroHash[:],
		StateRoot:     bytesutil.PadTo(signingRoot, 32),
		BodyRoot:      params.BeaconConfig().ZeroHash[:],
	}
	signRoot, err := header.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	return &slashertypes.SignedBlockHeaderWrapper{
		SignedBeaconBlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header:    header,
			Signature: params.BeaconConfig().EmptySignature[:],
		},
		SigningRoot: signRoot,
	}
}