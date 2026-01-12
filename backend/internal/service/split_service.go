package service

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/calculator"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// SplitService implements the Connect SplitService
type SplitService struct {
	protoconnect.UnimplementedSplitServiceHandler

	// Add storage layer when needed
	// store storage.Store
}

// NewSplitService creates a new SplitService
func NewSplitService() *SplitService {
	return &SplitService{}
}

// CalculateSplit handles bill split calculation
func (s *SplitService) CalculateSplit(ctx context.Context, req *connect.Request[pb.CalculateSplitRequest]) (*connect.Response[pb.CalculateSplitResponse], error) {
	slog.Info("CalculateSplit request received",
		"total", req.Msg.Total,
		"subtotal", req.Msg.Subtotal,
		"tax", req.Msg.Total-req.Msg.Subtotal,
		"participants", req.Msg.Participants,
		"items_count", len(req.Msg.Items),
	)

	// Convert proto items to calculator items
	items := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		slog.Debug("Processing item",
			"index", i+1,
			"description", item.Description,
			"amount", item.Amount,
			"assigned_to", item.AssignedTo,
		)
		items[i] = calculator.Item{
			Description: item.Description,
			Amount:      item.Amount,
			AssignedTo:  item.AssignedTo,
		}
	}

	splits, err := calculator.CalculateSplit(items, req.Msg.Total, req.Msg.Subtotal, req.Msg.Participants)
	if err != nil {
		slog.Error("CalculateSplit failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	slog.Info("Split calculation successful", "splits_count", len(splits))

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		slog.Debug("Person split",
			"person", person,
			"subtotal", split.Subtotal,
			"tax", split.Tax,
			"total", split.Total,
		)
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
		}
	}

	return connect.NewResponse(&pb.CalculateSplitResponse{
		Splits:    protoSplits,
		TaxAmount: req.Msg.Total - req.Msg.Subtotal,
		Subtotal:  req.Msg.Subtotal,
	}), nil
}

// CreateBill creates a new bill
func (s *SplitService) CreateBill(ctx context.Context, req *connect.Request[pb.CreateBillRequest]) (*connect.Response[pb.CreateBillResponse], error) {
	// TODO: Implement bill storage
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// GetBill retrieves a bill by ID
func (s *SplitService) GetBill(ctx context.Context, req *connect.Request[pb.GetBillRequest]) (*connect.Response[pb.GetBillResponse], error) {
	// TODO: Implement bill retrieval
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
