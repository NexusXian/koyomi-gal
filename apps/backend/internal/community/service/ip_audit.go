package service

import (
	"context"
	"strings"
	"time"

	"backend/internal/ipgeo"
	"backend/pkg/logger"

	"go.uber.org/zap"
)

const PermissionIPAuditRead = "ip_audit:read"

type IPLogEnqueuer interface {
	EnqueueIPLog(ctx context.Context, entry ipgeo.UserIPLog) error
}

func resolveContentIP(ctx context.Context, resolver *ipgeo.Service, clientIP string) (string, ipgeo.Location) {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return "", ipgeo.Location{}
	}
	if resolver == nil {
		return clientIP, ipgeo.Location{}
	}
	address, location, err := resolver.Resolve(ctx, clientIP)
	if err != nil {
		logger.Warn("resolve content IP location", zap.Error(err))
		return "", ipgeo.Location{}
	}
	return address, location
}

func enqueueContentIPLog(
	ctx context.Context,
	enqueuer IPLogEnqueuer,
	userID uint,
	action string,
	entityID uint,
	address string,
	location ipgeo.Location,
	createdAt time.Time,
) {
	if enqueuer == nil || address == "" {
		return
	}
	if err := enqueuer.EnqueueIPLog(ctx, ipgeo.UserIPLog{
		UserID: userID, IPAddress: address,
		Country: location.Country, Region: location.Region, City: location.City, ISP: location.ISP,
		Action: action, EntityID: &entityID, CreatedAt: createdAt,
	}); err != nil {
		logger.Error("enqueue content IP audit", zap.String("action", action), zap.Uint("entity_id", entityID), zap.Error(err))
	}
}
