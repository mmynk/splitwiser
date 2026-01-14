package service

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/mmynk/splitwiser/internal/calculator"
	"github.com/mmynk/splitwiser/internal/models"
	"github.com/mmynk/splitwiser/internal/storage"
	pb "github.com/mmynk/splitwiser/pkg/proto"
	"github.com/mmynk/splitwiser/pkg/proto/protoconnect"
)

// SplitService implements the Connect SplitService
type SplitService struct {
	protoconnect.UnimplementedSplitServiceHandler
	store storage.Store
}

// NewSplitService creates a new SplitService with the given storage backend.
func NewSplitService(store storage.Store) *SplitService {
	return &SplitService{store: store}
}

// validatePayerID checks if the payer is one of the participants.
func validatePayerID(payerID string, participants []string) error {
	if payerID == "" {
		return nil // Optional field
	}
	for _, p := range participants {
		if p == payerID {
			return nil
		}
	}
	return fmt.Errorf("payer_id '%s' must be one of the participants", payerID)
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
			"participants", item.Participants,
		)
		items[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
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
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		slog.Debug("Person split",
			"person", person,
			"subtotal", split.Subtotal,
			"tax", split.Tax,
			"total", split.Total,
			"items_count", len(split.Items),
		)
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.CalculateSplitResponse{
		Splits:    protoSplits,
		TaxAmount: req.Msg.Total - req.Msg.Subtotal,
		Subtotal:  req.Msg.Subtotal,
	}), nil
}

// CreateBill creates a new bill and persists it to storage.
func (s *SplitService) CreateBill(ctx context.Context, req *connect.Request[pb.CreateBillRequest]) (*connect.Response[pb.CreateBillResponse], error) {
	slog.Info("CreateBill request received",
		"title", req.Msg.Title,
		"total", req.Msg.Total,
		"subtotal", req.Msg.Subtotal,
		"participants", req.Msg.Participants,
		"items_count", len(req.Msg.Items),
	)

	// Convert proto items to models
	items := make([]models.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = models.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	// Validate payer
	if err := validatePayerID(req.Msg.GetPayerId(), req.Msg.Participants); err != nil {
		slog.Error("CreateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Create bill model
	bill := &models.Bill{
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: req.Msg.Participants,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	// Save to storage (generates ID and CreatedAt)
	if err := s.store.CreateBill(ctx, bill); err != nil {
		slog.Error("CreateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slog.Info("Bill created", "bill_id", bill.ID)

	// Calculate splits
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, req.Msg.Participants)
	if err != nil {
		slog.Error("CalculateSplit failed during CreateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.CreateBillResponse{
		BillId: bill.ID,
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: req.Msg.Total - req.Msg.Subtotal,
			Subtotal:  req.Msg.Subtotal,
		},
	}), nil
}

// GetBill retrieves a bill by ID from storage.
func (s *SplitService) GetBill(ctx context.Context, req *connect.Request[pb.GetBillRequest]) (*connect.Response[pb.GetBillResponse], error) {
	slog.Info("GetBill request received", "bill_id", req.Msg.BillId)

	// Retrieve from storage
	bill, err := s.store.GetBill(ctx, req.Msg.BillId)
	if err != nil {
		slog.Error("GetBill failed", "bill_id", req.Msg.BillId, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Convert items to proto format
	protoItems := make([]*pb.Item, len(bill.Items))
	for i, item := range bill.Items {
		protoItems[i] = &pb.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	// Recalculate splits
	calcItems := make([]calculator.Item, len(bill.Items))
	for i, item := range bill.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, bill.Total, bill.Subtotal, bill.Participants)
	if err != nil {
		slog.Error("CalculateSplit failed during GetBill", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		personItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			personItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    personItems,
		}
	}

	slog.Info("GetBill successful", "bill_id", bill.ID, "title", bill.Title)

	resp := &pb.GetBillResponse{
		BillId:       bill.ID,
		Title:        bill.Title,
		Items:        protoItems,
		Total:        bill.Total,
		Subtotal:     bill.Subtotal,
		Participants: bill.Participants,
		PayerId:      bill.PayerID,  // Now non-optional
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: bill.Total - bill.Subtotal,
			Subtotal:  bill.Subtotal,
		},
		CreatedAt: bill.CreatedAt,
	}
	if bill.GroupID != "" {
		resp.GroupId = &bill.GroupID
	}
	return connect.NewResponse(resp), nil
}

// UpdateBill updates an existing bill.
func (s *SplitService) UpdateBill(ctx context.Context, req *connect.Request[pb.UpdateBillRequest]) (*connect.Response[pb.UpdateBillResponse], error) {
	slog.Info("UpdateBill request received",
		"bill_id", req.Msg.BillId,
		"title", req.Msg.Title,
		"total", req.Msg.Total,
		"subtotal", req.Msg.Subtotal,
		"participants", req.Msg.Participants,
		"items_count", len(req.Msg.Items),
	)

	// Convert proto items to models
	items := make([]models.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		items[i] = models.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	// Validate payer
	if err := validatePayerID(req.Msg.GetPayerId(), req.Msg.Participants); err != nil {
		slog.Error("UpdateBill payer validation failed", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Create bill model
	bill := &models.Bill{
		ID:           req.Msg.BillId,
		Title:        req.Msg.Title,
		Items:        items,
		Total:        req.Msg.Total,
		Subtotal:     req.Msg.Subtotal,
		Participants: req.Msg.Participants,
	}
	if req.Msg.GetGroupId() != "" {
		bill.GroupID = req.Msg.GetGroupId()
	}
	if req.Msg.GetPayerId() != "" {
		bill.PayerID = req.Msg.GetPayerId()
	}

	// Update in storage
	if err := s.store.UpdateBill(ctx, bill); err != nil {
		slog.Error("UpdateBill failed", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	slog.Info("Bill updated", "bill_id", bill.ID)

	// Calculate splits
	calcItems := make([]calculator.Item, len(req.Msg.Items))
	for i, item := range req.Msg.Items {
		calcItems[i] = calculator.Item{
			Description:  item.Description,
			Amount:       item.Amount,
			Participants: item.Participants,
		}
	}

	splits, err := calculator.CalculateSplit(calcItems, req.Msg.Total, req.Msg.Subtotal, req.Msg.Participants)
	if err != nil {
		slog.Error("CalculateSplit failed during UpdateBill", "error", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert splits to proto format
	protoSplits := make(map[string]*pb.PersonSplit)
	for person, split := range splits {
		// Convert items to proto format
		protoItems := make([]*pb.PersonItem, len(split.Items))
		for i, item := range split.Items {
			protoItems[i] = &pb.PersonItem{
				Description: item.Description,
				Amount:      item.Amount,
			}
		}
		protoSplits[person] = &pb.PersonSplit{
			Subtotal: split.Subtotal,
			Tax:      split.Tax,
			Total:    split.Total,
			Items:    protoItems,
		}
	}

	return connect.NewResponse(&pb.UpdateBillResponse{
		BillId: bill.ID,
		Split: &pb.CalculateSplitResponse{
			Splits:    protoSplits,
			TaxAmount: req.Msg.Total - req.Msg.Subtotal,
			Subtotal:  req.Msg.Subtotal,
		},
	}), nil
}

// ListBillsByGroup retrieves all bills associated with a group.
func (s *SplitService) ListBillsByGroup(ctx context.Context, req *connect.Request[pb.ListBillsByGroupRequest]) (*connect.Response[pb.ListBillsByGroupResponse], error) {
	slog.Info("ListBillsByGroup request received", "group_id", req.Msg.GroupId)

	// Retrieve bills from storage
	bills, err := s.store.ListBillsByGroup(ctx, req.Msg.GroupId)
	if err != nil {
		slog.Error("ListBillsByGroup failed", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to bill summaries
	summaries := make([]*pb.BillSummary, len(bills))
	for i, bill := range bills {
		summaries[i] = &pb.BillSummary{
			BillId:           bill.ID,
			Title:            bill.Title,
			Total:            bill.Total,
			PayerId:          bill.PayerID,  // Now non-optional
			CreatedAt:        bill.CreatedAt,
			ParticipantCount: int32(len(bill.Participants)),
		}
	}

	slog.Info("ListBillsByGroup successful", "group_id", req.Msg.GroupId, "count", len(bills))

	return connect.NewResponse(&pb.ListBillsByGroupResponse{
		Bills: summaries,
	}), nil
}
