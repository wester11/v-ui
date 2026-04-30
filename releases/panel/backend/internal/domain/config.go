package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ConfigSetupMode string

const (
	ConfigSetupSimple   ConfigSetupMode = "simple"
	ConfigSetupAdvanced ConfigSetupMode = "advanced"
)

type ConfigRoutingMode string

const (
	ConfigRoutingSimple   ConfigRoutingMode = "simple"
	ConfigRoutingAdvanced ConfigRoutingMode = "advanced"
	ConfigRoutingCascade  ConfigRoutingMode = "cascade"
)

type ConfigTemplate string

const (
	TemplateVLESSReality ConfigTemplate = "vless_reality"
	TemplateGRPCReality  ConfigTemplate = "grpc_reality"
	TemplateCascade      ConfigTemplate = "cascade"
	TemplateEmpty        ConfigTemplate = "empty"
)

type VPNConfig struct {
	ID          uuid.UUID
	ServerID    uuid.UUID
	Name        string
	Protocol    Protocol
	Template    ConfigTemplate
	SetupMode   ConfigSetupMode
	RoutingMode ConfigRoutingMode
	Settings    json.RawMessage
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

