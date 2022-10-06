package stages

import (
	"context"

	"github.com/prebid/prebid-server/hooks/invocation"
)

type RawAuctionHook interface {
	Call(
		context.Context,
		invocation.Context,
		BidRequest,
	) (invocation.HookResult[BidRequest], error)
}

type BidRequest []byte
