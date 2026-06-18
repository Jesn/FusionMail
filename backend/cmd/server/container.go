package main

import (
	"fmt"
	"time"

	"fusionmail/config"
	"fusionmail/internal/adapter"
	"fusionmail/internal/handler"
	"fusionmail/internal/middleware"
	"fusionmail/internal/receiver"
	"fusionmail/internal/repository"
	"fusionmail/internal/router"
	"fusionmail/internal/service"
	"fusionmail/internal/service/spam"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/oauth2config"
	redisWrapper "fusionmail/pkg/redis"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AppContainer struct {
	Router  *gin.Engine
	Runtime *RuntimeResources
}

type RuntimeResources struct {
	SyncManager    *service.SyncManager
	CleanupService *service.CleanupService
}

func buildAppContainer(cfg *config.Config, db *gorm.DB, logDir string) (*AppContainer, error) {
	accountRepo := repository.NewAccountRepository(db)
	emailRepo := repository.NewEmailRepository(db)
	ruleRepo := repository.NewRuleRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)
	webhookLogRepo := repository.NewWebhookLogRepository(db)
	syncLogRepo := repository.NewSyncLogRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	providerRepo := repository.NewProviderRepository(db)
	oauth2ClientRepo := repository.NewOAuth2ClientRepository(db)
	adapterRepo := repository.NewAdapterRepository(db)
	deletedKeyRepo := repository.NewDeletedEmailKeyRepository(db)
	adapterFactory := adapter.NewFactory()

	cryptoService, err := crypto.NewService(cfg.Security.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("加密服务创建失败: %w", err)
	}
	oauth2ConfigProvider := oauth2config.NewProvider(oauth2ClientRepo, providerRepo, cryptoService, logger.NewWithModule("OAuth2Config"))
	credentialResolver := service.NewCredentialResolver(cryptoService, oauth2ConfigProvider)
	syncNotifier := service.NewSSESyncNotifier()

	accountService, err := service.NewAccountService(accountRepo, emailRepo, providerRepo, adapterFactory, cryptoService)
	if err != nil {
		return nil, fmt.Errorf("账户服务创建失败: %w", err)
	}

	encryptor, err := crypto.NewEncryptor()
	if err != nil {
		return nil, fmt.Errorf("加密器创建失败: %w", err)
	}

	emailService := service.NewEmailServiceWithCredentialResolver(emailRepo, accountRepo, adapterFactory, credentialResolver)
	translationService := service.NewTranslationService(service.TranslationServiceConfig{
		APIURL:  cfg.Translation.APIURL,
		Token:   cfg.Translation.Token,
		Timeout: time.Duration(cfg.Translation.TimeoutSeconds) * time.Second,
	})
	ruleService := service.NewRuleService(ruleRepo, emailRepo)

	if err := redisWrapper.Initialize(&cfg.Redis); err != nil {
		log.Warn("Redis 连接失败: %v", err)
	}
	redisClient := redisWrapper.GetClient()
	redisClientWrapper := redisWrapper.NewClientWrapper(redisClient)

	webhookLogger := logger.New()
	webhookService := service.NewWebhookService(webhookRepo, webhookLogRepo, webhookLogger)
	oauth2Service := service.NewOAuth2Service(cfg, accountRepo, emailRepo, cryptoService, redisClientWrapper, webhookLogger, oauth2ClientRepo, providerRepo, adapterRepo)
	systemService := service.NewSystemService(
		db,
		redisClient,
		accountRepo,
		emailRepo,
		ruleRepo,
		webhookRepo,
		syncLogRepo,
		providerRepo,
		webhookLogger,
	)
	oauth2ClientService := service.NewOAuth2ClientService(oauth2ClientRepo, cryptoService)
	providerService := service.NewProviderService(providerRepo)
	webAPIProviderService := service.NewWebAPIProviderService(
		accountRepo,
		providerRepo,
		adapterRepo,
		emailRepo,
		syncLogRepo,
		cryptoService,
	)
	adapterService := service.NewAdapterService(adapterRepo)

	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, apiKeyRepo, cfg.JWT.Secret, time.Duration(cfg.JWT.Expiry)*time.Hour)

	jwtSecret := cfg.JWT.Secret
	authHandler := handler.NewDBAuthHandler(jwtSecret, cfg.Security.CookieSecure)
	emailHandler := handler.NewEmailHandler(emailService)
	translationHandler := handler.NewTranslationHandler(translationService)
	ruleHandler := handler.NewRuleHandler(ruleService)
	webhookHandler := handler.NewWebhookHandler(webhookService, webhookLogRepo)
	systemHandler := handler.NewSystemHandler(systemService)
	oauth2Handler := handler.NewOAuth2Handler(oauth2Service)
	apiKeyHandler := handler.NewAPIKeyHandler(authService)
	oauth2ClientHandler := handler.NewOAuth2ClientHandler(oauth2ClientService, providerService)
	providerHandler := handler.NewProviderHandler(providerService)
	adapterHandler := handler.NewAdapterHandler(adapterService)
	devSyncHandler := handler.NewDevSyncHandler()
	webAPIProviderHandler := handler.NewWebAPIProviderHandler(webAPIProviderService)
	webAPIServicesHandler := handler.NewWebAPIServicesHandler()

	settingService := service.NewSettingService(settingRepo, redisClient, encryptor)
	settingHandler := handler.NewSettingHandler(settingService)

	groupRepo := repository.NewGroupRepository(db)
	groupService := service.NewGroupService(groupRepo, accountRepo)
	groupHandler := handler.NewGroupHandler(groupService)

	emailListRepo := repository.NewEmailListRepository(db)
	whitelistChecker := spam.NewWhitelistChecker(emailListRepo, redisClient)
	emailListService := spam.NewEmailListService(emailListRepo, whitelistChecker)
	emailListHandler := handler.NewEmailListHandler(emailListService)

	senderReputationRepo := repository.NewSenderReputationRepository(db)
	bayesianTrainingRepo := repository.NewBayesianTrainingRepository(db)
	spamRuleRepo := repository.NewSpamRuleRepository(db)
	reputationManager := spam.NewReputationManager(senderReputationRepo, redisClient)
	bayesianClassifier := spam.NewBayesianClassifier(bayesianTrainingRepo)
	spamService := service.NewSpamService(emailRepo, spamRuleRepo, reputationManager, bayesianClassifier)
	spamHandler := handler.NewSpamHandler(spamService)
	reputationHandler := handler.NewReputationHandler(reputationManager, senderReputationRepo)

	sentEmailRepo := repository.NewSentEmailRepository(db)
	senderFactory, err := adapter.NewSenderFactory(cfg.Security.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("发送器工厂创建失败: %w", err)
	}
	sendService := service.NewSendService(senderFactory, accountRepo, sentEmailRepo, emailRepo, cryptoService, oauth2ConfigProvider, logger.NewWithModule("SendService"))
	sentEmailService := service.NewSentEmailService(sentEmailRepo)
	smtpConfigService, err := service.NewSMTPConfigService(accountRepo, cfg.Security.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("SMTP 配置服务创建失败: %w", err)
	}
	sendHandler := handler.NewSendHandler(sendService, sentEmailService, smtpConfigService)

	spamDetectionLogRepo := repository.NewSpamDetectionLogRepository(db)
	rblChecker := spam.NewRBLChecker(redisClient)
	behaviorAnalyzer := spam.NewBehaviorAnalyzer(redisClient)
	surblChecker := spam.NewSURBLChecker(redisClient)
	preFilter := spam.NewPreFilter(rblChecker, behaviorAnalyzer)
	ruleEngine := spam.NewRuleEngine(spamRuleRepo, redisClient, surblChecker)
	spamDetector := spam.NewSpamDetector(whitelistChecker, preFilter, ruleEngine, reputationManager, bayesianClassifier, spamDetectionLogRepo)

	syncService := service.NewSyncService(
		accountRepo,
		emailRepo,
		syncLogRepo,
		deletedKeyRepo,
		adapterFactory,
		webhookLogger,
		cryptoService,
		credentialResolver,
		ruleService,
		spamDetector,
		redisClient,
		syncNotifier,
	)
	syncManager := service.NewSyncManagerWithDeps(syncService, accountRepo, adapterFactory, credentialResolver)

	accountHandler := handler.NewAccountHandler(accountService, oauth2Service, syncManager.GetSyncService())
	publicHandler := handler.NewPublicHandler(emailService, accountService, syncManager.GetSyncService())

	cleanupService := service.NewCleanupService(
		accountService,
		settingService,
		emailRepo,
		syncLogRepo,
		webhookLogRepo,
		spamDetectionLogRepo,
		deletedKeyRepo,
	)

	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)
	routerDeps := router.RouterDeps{
		Handlers: router.Handlers{
			Auth:         authHandler,
			Account:      accountHandler,
			Email:        emailHandler,
			Rule:         ruleHandler,
			Webhook:      webhookHandler,
			System:       systemHandler,
			OAuth2:       oauth2Handler,
			APIKey:       apiKeyHandler,
			Public:       publicHandler,
			Setting:      settingHandler,
			OAuth2Client: oauth2ClientHandler,
			Provider:     providerHandler,
			Adapter:      adapterHandler,
			DevSync:      devSyncHandler,
			EmailList:    emailListHandler,
			Spam:         spamHandler,
			Reputation:   reputationHandler,
		},
		SyncManager:    syncManager,
		RedisClient:    redisClient,
		JWTSecret:      jwtSecret,
		CookieSecure:   cfg.Security.CookieSecure,
		AuthMiddleware: authMiddleware,
		APIKeyRepo:     apiKeyRepo,
		RateLimit: router.RateLimitConfig{
			Enabled:      cfg.RateLimit.Enabled,
			SitePerMin:   cfg.RateLimit.SiteDefault,
			PublicPerMin: cfg.RateLimit.PublicDefault,
		},
	}

	ginRouter := router.SetupRouter(routerDeps)

	router.RegisterGroupRoutes(ginRouter, groupHandler, routerDeps)
	log.Info("分组管理路由已注册")

	router.RegisterSendRoutes(ginRouter, sendHandler, routerDeps)
	log.Info("邮件发送路由已注册")
	router.RegisterTranslationRoutes(ginRouter, translationHandler, routerDeps)

	router.RegisterWebAPIRoutes(ginRouter, webAPIProviderHandler, webAPIServicesHandler, routerDeps)
	log.Info("WebAPI Provider 路由已注册")

	webhookRegistry := receiver.NewAdapterRegistry()
	webhookRegistry.Register(receiver.NewCloudflareAdapter(webhookLogger))
	log.Info("Webhook 适配器已注册: %v", webhookRegistry.List())

	webhookReceiverService := service.NewWebhookReceiverServiceWithNotifier(accountRepo, emailRepo, providerRepo, cryptoService, webhookLogger, syncNotifier)
	webhookReceiverHandler := handler.NewWebhookReceiverHandler(webhookRegistry, webhookReceiverService, webhookLogger)
	router.RegisterWebhookReceiverRoutes(ginRouter, webhookReceiverHandler)
	log.Info("Webhook 接收路由已注册")

	logHandler := handler.NewLogHandler(logDir)
	router.RegisterLogRoutes(ginRouter, logHandler, routerDeps)
	log.Info("日志查询路由已注册，日志目录: %s", logDir)

	return &AppContainer{
		Router: ginRouter,
		Runtime: &RuntimeResources{
			SyncManager:    syncManager,
			CleanupService: cleanupService,
		},
	}, nil
}
