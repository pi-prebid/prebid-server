package hooks

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookstage"
)

type EmptyPlanBuilder struct{}

func (e EmptyPlanBuilder) PlanForEntrypointStage(endpoint string) Plan[hookstage.Entrypoint] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawAuctionStage(endpoint string, account *config.Account) Plan[hookstage.RawAuction] {
	return nil
}

func (e EmptyPlanBuilder) PlanForProcessedAuctionStage(endpoint string, account *config.Account) Plan[hookstage.ProcessedAuction] {
	return nil
}

func (e EmptyPlanBuilder) PlanForBidderRequestStage(endpoint string, account *config.Account) Plan[hookstage.BidderRequest] {
	return nil
}

func (e EmptyPlanBuilder) PlanForRawBidderResponseStage(endpoint string, account *config.Account) Plan[hookstage.RawBidderResponse] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAllProcessedBidResponsesStage(endpoint string, account *config.Account) Plan[hookstage.AllProcessedBidResponses] {
	return nil
}

func (e EmptyPlanBuilder) PlanForAuctionResponseStage(endpoint string, account *config.Account) Plan[hookstage.AuctionResponse] {
	return nil
}
