package constans

const (
	StaticDefaultEnv = "app.env.default"
	UserCache        = "app.user.cache"
	UpdateBySystem   = "SYSTEM"
	TbKey            = "app.api.key.tb.credential.secret"
	WaAccountId      = "app.api.key.wa.account-id"
	WaBaId           = "app.api.key.wa.waba-id"
	AppId            = "app.api.key.wa.app-id"
	WaPhoneNumberId  = "app.api.key.wa.account-id"
	WaAuthToken      = "app.api.key.wa.access-token"
	GroqApiKey       = "app.api.key.groq"
	GroqAIName       = "app.ai.name"
	AIDefaultSender  = "6285591523647" // Static sender for AI auto-response (temporary)
	WaClientId       = "app.api.key.wa.client-id"
	WaClientSecret   = "app.api.key.wa.client-secret"

	BaseURLMeta = "app.base.url.meta"

	WaWebHookWorkerNum     = "app.workers.default"
	WaPrefixName           = "dashboard:"
	WaWebHookWorkerSubName = WaPrefixName + "wa:webhook"

	// AI Cache Keys
	AIKnowledgeCachePrefix      = "ai:knowledge:client:"
	AIKnowledgeCacheLastRefresh = "ai:knowledge:last-refresh:"

	// PPOB API Configuration
	PPOBSecretCode    = "app.payment.secret-code"
	PPOBLoginEmail    = "app.payment.login.email"
	PPOBLoginPassword = "app.payment.login.password"
	PPOBProductCode   = "app.payment.client.product-code"
	PPOBMerchantUser  = "app.payment.merchant.user"
	PPOBMerchantCode  = "app.payment.merchant.code"
	PPOBCategory      = "app.payment.category"
	PPOBMethod        = "app.payment.method"
	PPOBBaseURL       = "app.payment.base-url"
	PPOBAuthURL       = "app.payment.auth-url"

	// PPOB Cache Keys
	PPOBAccessTokenCache = "ppob:access-token"

	SingleSendMessageSubscribe = "single:send:message:subscribe"

	BillingTransactionTypeTopup      = "topup"
	BillingTransactionTypeUsage      = "usage"
	BillingTransactionTypePayment    = "payment"
	BillingTransactionTypeRefund     = "refund"
	BillingTransactionTypeAdjustment = "adjustment"
)
