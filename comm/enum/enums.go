package enum

const (
	UserTypeStudent     = "STUDENT"
	UserTypeAdmin       = "ADMIN"
	UserTypeSystemAdmin = "SYSTEM_ADMIN"
)

const (
	PostTypeLost  = "LOST"
	PostTypeFound = "FOUND"
)

const (
	PostStatusPending   = "PENDING"
	PostStatusApproved  = "APPROVED"
	PostStatusMatched   = "MATCHED"
	PostStatusClaimed   = "CLAIMED"
	PostStatusCancelled = "CANCELLED"
	PostStatusRejected  = "REJECTED"
	PostStatusArchived  = "ARCHIVED"
)

const (
	AuditLogTypeLogin  = "LOGIN"
	AuditLogTypeCreate = "CREATE"
	AuditLogTypeUpdate = "UPDATE"
	AuditLogTypeDelete = "DELETE"
)

const (
	CampusZhaoHui   = "ZHAO_HUI"
	CampusPingFeng  = "PING_FENG"
	CampusMoGanShan = "MO_GAN_SHAN"
)

const (
	ClaimStatusPending  = "PENDING"
	ClaimStatusMatched  = "MATCHED"
	ClaimStatusRejected = "REJECTED"
)

const (
	AnnouncementTypeSystem = "SYSTEM"
	AnnouncementTypeRegion = "REGION"
)

const (
	AnnouncementStatusPending  = "PENDING"
	AnnouncementStatusApproved = "APPROVED"
)
