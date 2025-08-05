package services

import (
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/repository"
)

// Services holds all service instances
type Services struct {
	Chat        *ChatService
	Config      *ConfigService
	Providers   *providers.Registry
	UnifiedChat *UnifiedChatService
	Router      *RequestRouter
	Connection  *ConnectionService
}

// NewServices creates all service instances
func NewServices(
	providers *providers.Registry,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	configRepo repository.ConfigRepository,
	connectionRepo repository.ConnectionRepository,
) *Services {
	configService := NewConfigService(configRepo)
	router := NewRequestRouter(providers, configService)
	connectionService := NewConnectionService(connectionRepo, providers)
	
	return &Services{
		Chat:        NewChatService(providers, sessionRepo, messageRepo),
		Config:      configService,
		Providers:   providers,
		UnifiedChat: NewUnifiedChatService(providers, configService, sessionRepo, messageRepo),
		Router:      router,
		Connection:  connectionService,
	}
}