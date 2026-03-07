package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewIdeaService, NewAuthService, NewQueueService, NewRegistryService)
