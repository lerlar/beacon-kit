// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2024, Berachain Foundation. All rights reserved.
// Use of this software is governed by the Business Source License included
// in the LICENSE file of this repository and at www.mariadb.com/bsl11.
//
// ANY USE OF THE LICENSED WORK IN VIOLATION OF THIS LICENSE WILL AUTOMATICALLY
// TERMINATE YOUR RIGHTS UNDER THIS LICENSE FOR THE CURRENT AND ALL OTHER
// VERSIONS OF THE LICENSED WORK.
//
// THIS LICENSE DOES NOT GRANT YOU ANY RIGHT IN ANY TRADEMARK OR LOGO OF
// LICENSOR OR ITS AFFILIATES (PROVIDED THAT YOU MAY USE A TRADEMARK OR LOGO OF
// LICENSOR AS EXPRESSLY REQUIRED BY THIS LICENSE).
//
// TO THE EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSED WORK IS PROVIDED ON
// AN “AS IS” BASIS. LICENSOR HEREBY DISCLAIMS ALL WARRANTIES AND CONDITIONS,
// EXPRESS OR IMPLIED, INCLUDING (WITHOUT LIMITATION) WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, NON-INFRINGEMENT, AND
// TITLE.

package middleware

import (
	"context"

	"github.com/berachain/beacon-kit/mod/async/pkg/event"
	asynctypes "github.com/berachain/beacon-kit/mod/async/pkg/types"
	"github.com/berachain/beacon-kit/mod/log"
	"github.com/berachain/beacon-kit/mod/p2p"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/common"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/events"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/math"
	"github.com/berachain/beacon-kit/mod/primitives/pkg/ssz"
	"github.com/berachain/beacon-kit/mod/runtime/pkg/encoding"
	rp2p "github.com/berachain/beacon-kit/mod/runtime/pkg/p2p"
	cmtabci "github.com/cometbft/cometbft/abci/types"
)

// ABCIMiddleware is a middleware between ABCI and the validator logic.
type ABCIMiddleware[
	AvailabilityStoreT any,
	BeaconBlockT BeaconBlock[BeaconBlockT],
	BeaconStateT BeaconState,
	BlobSidecarsT ssz.Marshallable,
	DepositT,
	ExecutionPayloadT any,
	GenesisT Genesis,
] struct {
	// chainSpec is the chain specification.
	chainSpec common.ChainSpec
	// chainService represents the blockchain service.
	chainService BlockchainService[
		BeaconBlockT, BlobSidecarsT, DepositT, GenesisT,
	]
	// daService is the service responsible for building the data availability
	daService DAService[BlobSidecarsT]
	// TODO: we will eventually gossip the blobs separately from
	// CometBFT, but for now, these are no-op gossipers.
	blobGossiper p2p.PublisherReceiver[
		BlobSidecarsT,
		[]byte,
		encoding.ABCIRequest,
		BlobSidecarsT,
	]
	// TODO: we will eventually gossip the blocks separately from
	// CometBFT, but for now, these are no-op gossipers.
	beaconBlockGossiper p2p.PublisherReceiver[
		BeaconBlockT,
		[]byte,
		encoding.ABCIRequest,
		BeaconBlockT,
	]
	// metrics is the metrics emitter.
	metrics *ABCIMiddlewareMetrics
	// logger is the logger for the middleware.
	logger log.Logger[any]

	// Feeds
	//
	// blkFeed is a feed for blocks.
	blkFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[BeaconBlockT]]
	// sidecarsFeed is a feed for sidecars.
	sidecarsFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[BlobSidecarsT]]
	// slotFeed is a feed for slots.
	slotFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[math.Slot]]

	// TODO: this is a temporary hack.
	req *cmtabci.FinalizeBlockRequest

	// Channels
	//
	// PrepareProposal
	//
	// errCh is used to communicate errors to the EndBlock
	// method.
	errCh chan error
	// blkCh is used to communicate the beacon block to the EndBlock method.
	blkCh chan *asynctypes.Event[BeaconBlockT]
	// sidecarsCh is used to communicate the sidecars to the EndBlock method.
	sidecarsCh chan *asynctypes.Event[BlobSidecarsT]
}

// NewABCIMiddleware creates a new instance of the Handler struct.
func NewABCIMiddleware[
	AvailabilityStoreT any,
	BeaconBlockT BeaconBlock[BeaconBlockT],
	BeaconStateT BeaconState,
	BlobSidecarsT ssz.Marshallable,
	DepositT,
	ExecutionPayloadT any,
	GenesisT Genesis,
](
	chainSpec common.ChainSpec,
	chainService BlockchainService[
		BeaconBlockT, BlobSidecarsT, DepositT, GenesisT,
	],
	daService DAService[BlobSidecarsT],
	logger log.Logger[any],
	telemetrySink TelemetrySink,
	blkFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[BeaconBlockT]],
	sidecarsFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[BlobSidecarsT]],
	slotFeed *event.FeedOf[
		asynctypes.EventID, *asynctypes.Event[math.Slot]],
) *ABCIMiddleware[
	AvailabilityStoreT, BeaconBlockT, BeaconStateT,
	BlobSidecarsT, DepositT, ExecutionPayloadT, GenesisT,
] {
	return &ABCIMiddleware[
		AvailabilityStoreT, BeaconBlockT, BeaconStateT,
		BlobSidecarsT, DepositT, ExecutionPayloadT, GenesisT,
	]{
		chainSpec:    chainSpec,
		chainService: chainService,
		daService:    daService,
		blobGossiper: rp2p.NewNoopBlobHandler[
			BlobSidecarsT, encoding.ABCIRequest](),
		beaconBlockGossiper: rp2p.
			NewNoopBlockGossipHandler[BeaconBlockT, encoding.ABCIRequest](
			chainSpec,
		),
		logger:       logger,
		metrics:      newABCIMiddlewareMetrics(telemetrySink),
		blkFeed:      blkFeed,
		sidecarsFeed: sidecarsFeed,
		slotFeed:     slotFeed,
		blkCh: make(
			chan *asynctypes.Event[BeaconBlockT],
			1,
		),
		sidecarsCh: make(
			chan *asynctypes.Event[BlobSidecarsT],
			1,
		),
		errCh: make(chan error, 1),
	}
}

// Name returns the name of the middleware.
func (am *ABCIMiddleware[
	AvailabilityStoreT, BeaconBlockT, BeaconStateT,
	BlobSidecarsT, DepositT, ExecutionPayloadT, GenesisT,
]) Name() string {
	return "abci-middleware"
}

// Start the middleware.
func (am *ABCIMiddleware[
	_, _, _, _, _, _, _,
]) Start(ctx context.Context) error {
	go am.start(ctx)
	return nil
}

// start starts the middleware.
func (am *ABCIMiddleware[
	_, BeaconBlockT, _, BlobSidecarsT, _, _, _,
]) start(ctx context.Context) {
	subSidecarsCh := make(chan *asynctypes.Event[BlobSidecarsT], 1)
	subBlkCh := make(chan *asynctypes.Event[BeaconBlockT], 1)
	blkSub := am.blkFeed.Subscribe(subBlkCh)
	sidecarsSub := am.sidecarsFeed.Subscribe(subSidecarsCh)
	defer blkSub.Unsubscribe()
	defer sidecarsSub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-subBlkCh:
			switch msg.Type() {
			case events.BeaconBlockBuilt:
				fallthrough
			case events.BeaconBlockVerified:
				am.blkCh <- msg
			}
		case msg := <-subSidecarsCh:
			switch msg.Type() {
			case events.BlobSidecarsBuilt:
				fallthrough
			case events.BlobSidecarsVerified:
				fallthrough
			case events.BlobSidecarsProcessed:
				am.sidecarsCh <- msg
			}
		}
	}
}
