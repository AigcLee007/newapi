package model

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type startupLegacyToken struct {
	Id          int    `gorm:"primaryKey"`
	UserId      int    `gorm:"column:user_id;index"`
	Key         string `gorm:"type:char(48);uniqueIndex"`
	ModelLimits string `gorm:"type:varchar(1024)"`
}

func (startupLegacyToken) TableName() string { return "tokens" }

func TestMigrateDBStartupTwice(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db, dbType := openMigrationStartupDatabase(t, dialect)
			recorder := &migrationSQLRecorder{}
			db = db.Session(&gorm.Session{Logger: recorder})
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

			previousDB := DB
			previousLogDB := LOG_DB
			previousMainType := common.MainDatabaseType()
			previousLogType := common.LogDatabaseType()
			previousRedisEnabled := common.RedisEnabled
			previousRetireAt := LegacyAccessTokenRetireAt()
			DB, LOG_DB = db, db
			common.SetDatabaseTypes(dbType, dbType)
			common.RedisEnabled = false
			initCol()
			t.Cleanup(func() {
				DB, LOG_DB = previousDB, previousLogDB
				common.SetDatabaseTypes(previousMainType, previousLogType)
				common.RedisEnabled = previousRedisEnabled
				legacyAccessTokenRetireAt.Store(previousRetireAt)
				initCol()
			})

			dropMigrationStartupTables(t, db)
			require.NoError(t, migrateDB())
			require.NoError(t, db.Create(&Channel{Id: 29, Type: constant.ChannelTypeAdvancedCustom, Key: "fresh-official"}).Error)
			recorder.reset()
			require.NoError(t, migrateDB())
			require.Empty(t, recorder.schemaMutations(), "second fresh startup must not rewrite the schema")
			var fresh Channel
			require.NoError(t, db.First(&fresh, 29).Error)
			require.Equal(t, constant.ChannelTypeAdvancedCustom, fresh.Type)

			dropMigrationStartupTables(t, db)
			require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}, &Task{}, &Option{}, &ImageAsyncIdempotency{}, &startupLegacyToken{}))
			customURL := "https://custom-proxy.example/images"
			legacyChannel := Channel{Id: 27, Type: constant.ChannelTypeAdvancedCustom, Key: "legacy-secret", Name: "legacy alias", BaseURL: &customURL, Models: "custom-model"}
			require.NoError(t, db.Create(&legacyChannel).Error)
			require.NoError(t, db.Create(&Ability{ChannelId: legacyChannel.Id, Group: "default", Model: "custom-model", Enabled: true}).Error)
			require.NoError(t, db.Create(&Task{TaskID: "legacy-image-task", ChannelId: legacyChannel.Id, PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{RequestHash: "preserved-hash"}}}).Error)
			legacyKey := strings.Repeat("a", 48)
			require.NoError(t, db.Create(&startupLegacyToken{Id: 17, UserId: 3, Key: legacyKey, ModelLimits: "custom-model"}).Error)
			if dialect == "postgres" {
				for _, name := range []string{postgresTokenKeyConstraint, gormTokenKeyConstraint} {
					require.NoError(t, db.Exec("ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)", clause.Table{Name: "tokens"}, clause.Column{Name: name}, clause.Column{Name: "key"}).Error)
				}
			}

			require.NoError(t, migrateDB())
			var upgraded Channel
			require.NoError(t, db.First(&upgraded, legacyChannel.Id).Error)
			require.Equal(t, constant.ChannelTypeVisionary, upgraded.Type)
			require.Equal(t, legacyChannel.Key, upgraded.Key)
			require.Equal(t, legacyChannel.Name, upgraded.Name)
			require.Equal(t, legacyChannel.BaseURL, upgraded.BaseURL)
			require.Equal(t, legacyChannel.Models, upgraded.Models)
			var token Token
			require.NoError(t, db.First(&token, 17).Error)
			require.Equal(t, legacyKey, token.Key)
			require.Equal(t, "custom-model", token.ModelLimits)
			var keyLength int64
			for _, column := range mustColumnTypes(t, db, &Token{}) {
				if column.Name() == "key" {
					keyLength, _ = column.Length()
				}
			}
			require.EqualValues(t, 128, keyLength)
			require.True(t, db.Migrator().HasIndex(&Token{}, tokenKeyIndex))
			require.True(t, db.Migrator().HasIndex(&Task{}, db.NamingStrategy.IndexName("tasks", "channel_id")))
			require.Error(t, db.Create(&Token{Id: 18, UserId: 4, Key: legacyKey}).Error)

			require.NoError(t, db.Create(&Channel{Id: 30, Type: constant.ChannelTypeAdvancedCustom, Key: "post-upgrade-official"}).Error)
			recorder.reset()
			require.NoError(t, migrateDB())
			require.Empty(t, recorder.schemaMutations(), "second legacy startup must not rewrite the schema")
			upgraded = Channel{}
			require.NoError(t, db.First(&upgraded, legacyChannel.Id).Error)
			require.Equal(t, constant.ChannelTypeVisionary, upgraded.Type)
			var laterOfficial Channel
			require.NoError(t, db.First(&laterOfficial, 30).Error)
			require.Equal(t, constant.ChannelTypeAdvancedCustom, laterOfficial.Type)
			var savedTask Task
			require.NoError(t, db.Where("task_id = ?", "legacy-image-task").First(&savedTask).Error)
			require.Equal(t, legacyChannel.Id, savedTask.ChannelId)
			require.NotNil(t, savedTask.PrivateData.ImageAsync)
			require.Equal(t, "preserved-hash", savedTask.PrivateData.ImageAsync.RequestHash)
			var savedAbility Ability
			require.NoError(t, db.Where("channel_id = ?", legacyChannel.Id).First(&savedAbility).Error)
			require.Equal(t, "custom-model", savedAbility.Model)
			require.Error(t, db.Create(&savedAbility).Error, "ability uniqueness must survive")
			var marker Option
			require.NoError(t, db.Where(&Option{Key: legacyVisionaryChannelTypeMigrationKey}).First(&marker).Error)
			require.Equal(t, "1", marker.Value)
			dropMigrationStartupTables(t, db)
		})
	}
}

func TestImageAsyncRefundDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db, dbType := openMigrationStartupDatabase(t, dialect)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
			if dialect == "sqlite" {
				sqlDB.SetMaxOpenConns(1)
			}
			previousDB, previousType := DB, common.MainDatabaseType()
			previousRedis := common.RedisEnabled
			DB, common.RedisEnabled = db, false
			common.SetMainDatabaseType(dbType)
			initCol()
			t.Cleanup(func() {
				DB, common.RedisEnabled = previousDB, previousRedis
				common.SetMainDatabaseType(previousType)
				initCol()
			})
			require.NoError(t, db.AutoMigrate(&User{}, &Token{}, &Channel{}, &Task{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}))
			versionQuery := "SELECT version()"
			if dialect == "sqlite" {
				versionQuery = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
			t.Logf("%s version: %s", dialect, version)
			for _, source := range []string{"wallet", "subscription"} {
				t.Run("reservation_"+source, func(t *testing.T) {
					for caseIndex, scenario := range []string{"lifecycle", "concurrent", "insufficient_funding", "insufficient_token", "fail_tokens", "fail_tasks", "ownership_changed", "negative_target", "oversized_target"} {
						t.Run(scenario, func(t *testing.T) {
							label := fmt.Sprintf("reserve_%s_%d", source, caseIndex)
							initialWallet, initialToken, initialSubscription := 10000, 10000, int64(500)
							if scenario == "insufficient_funding" {
								if source == "wallet" {
									initialWallet = 500
								} else {
									initialSubscription = 9500
								}
							}
							if scenario == "insufficient_token" {
								initialToken = 500
							}
							user := User{Username: label, AffCode: label, Quota: initialWallet, UsedQuota: 700, RequestCount: 3}
							require.NoError(t, db.Create(&user).Error)
							token := Token{UserId: user.Id, Key: label, RemainQuota: initialToken, UsedQuota: 200}
							require.NoError(t, db.Create(&token).Error)
							channel := Channel{Name: label, UsedQuota: 700}
							require.NoError(t, db.Create(&channel).Error)
							sub := UserSubscription{UserId: user.Id, AmountTotal: 10000, AmountUsed: initialSubscription}
							require.NoError(t, db.Create(&sub).Error)
							task := Task{TaskID: label, UserId: user.Id, ChannelId: channel.Id, Status: TaskStatusInProgress, PrivateData: TaskPrivateData{BillingSource: source, SubscriptionId: sub.Id, TokenId: token.Id, ImageAsync: &ImageAsyncPrivateData{RequestHash: "preserved-reservation-hash", PriceDataJSON: []byte(`{"model_ratio":1}`)}}}
							require.NoError(t, db.Create(&task).Error)
							task.PrivateData.ImageAsync.PriceDataJSON = []byte(`{"model_ratio":2}`)
							task.PrivateData.ImageAsync.ImageQuotaBeforeGroup = 2000
							task.PrivateData.ImageAsync.BillingRequestInputJSON = []byte(`{"image_count":2}`)
							task.PrivateData.BillingContext = &TaskBillingContext{ModelPrice: 2, GroupRatio: 1.5}
							stale := task
							assertReservation := func(reserved int) Task {
								require.NoError(t, db.First(&user, user.Id).Error)
								require.NoError(t, db.First(&token, token.Id).Error)
								require.NoError(t, db.First(&channel, channel.Id).Error)
								require.NoError(t, db.First(&sub, sub.Id).Error)
								var stored Task
								require.NoError(t, db.First(&stored, task.ID).Error)
								wantWallet, wantSubscription := initialWallet, initialSubscription
								if source == "wallet" {
									wantWallet -= reserved
								} else {
									wantSubscription += int64(reserved)
								}
								assert.Equal(t, wantWallet, user.Quota)
								assert.Equal(t, wantSubscription, sub.AmountUsed)
								assert.Equal(t, initialToken-reserved, token.RemainQuota)
								assert.Equal(t, 200+reserved, token.UsedQuota)
								assert.Equal(t, reserved, stored.Quota)
								assert.Equal(t, 700, user.UsedQuota)
								assert.Equal(t, 3, user.RequestCount)
								assert.EqualValues(t, 700, channel.UsedQuota)
								require.NotNil(t, stored.PrivateData.ImageAsync)
								assert.Equal(t, "preserved-reservation-hash", stored.PrivateData.ImageAsync.RequestHash)
								return stored
							}
							switch scenario {
							case "fail_tokens", "fail_tasks":
								failTable := strings.TrimPrefix(scenario, "fail_")
								const callback = "image-reserve:fail-update"
								require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
									if tx.Statement.Table == failTable {
										_ = tx.AddError(errors.New("injected reservation write failure"))
									}
								}))
								func() {
									defer func() { require.NoError(t, db.Callback().Update().Remove(callback)) }()
									reserved, err := ReserveImageAsyncReservation(&task, 1000)
									require.ErrorContains(t, err, "injected reservation write failure")
									assert.False(t, reserved)
								}()
							case "insufficient_funding", "insufficient_token":
								reserved, err := ReserveImageAsyncReservation(&task, 1000)
								require.ErrorContains(t, err, "quota insufficient")
								assert.False(t, reserved)
							case "ownership_changed":
								require.NoError(t, db.Model(&task).Update("status", TaskStatusFailure).Error)
								reserved, err := ReserveImageAsyncReservation(&task, 1000)
								require.NoError(t, err)
								assert.False(t, reserved)
							case "negative_target", "oversized_target":
								target := -1
								if scenario == "oversized_target" {
									target = common.MaxQuota + 1
								}
								reserved, err := ReserveImageAsyncReservation(&task, target)
								require.ErrorContains(t, err, "reservation is out of range")
								assert.False(t, reserved)
							case "concurrent":
								type result struct {
									reserved bool
									err      error
								}
								results := make(chan result, 2)
								for range 2 {
									go func() {
										caller := stale
										reserved, err := ReserveImageAsyncReservation(&caller, 1000)
										results <- result{reserved: reserved, err: err}
									}()
								}
								for range 2 {
									result := <-results
									require.NoError(t, result.err)
									assert.True(t, result.reserved)
								}
								assertReservation(1000)
								return
							case "lifecycle":
								for _, step := range []struct{ target, want int }{{1000, 1000}, {1000, 1000}, {1500, 1500}, {500, 1500}} {
									reserved, err := ReserveImageAsyncReservation(&stale, step.target)
									require.NoError(t, err)
									assert.True(t, reserved)
									stored := assertReservation(step.want)
									assert.Equal(t, step.want, stale.Quota)
									assert.JSONEq(t, `{"model_ratio":2}`, string(stored.PrivateData.ImageAsync.PriceDataJSON))
									assert.Equal(t, float64(2000), stored.PrivateData.ImageAsync.ImageQuotaBeforeGroup)
									assert.JSONEq(t, `{"image_count":2}`, string(stored.PrivateData.ImageAsync.BillingRequestInputJSON))
									assert.Equal(t, task.PrivateData.BillingContext, stored.PrivateData.BillingContext)
									reserved, err = ReserveImageAsyncReservation(&stored, step.target)
									require.NoError(t, err)
									assert.True(t, reserved)
									assertReservation(step.want)
									stale.Quota = 0 // An older caller must still use the locked database amount.
								}
								refunded, err := RefundImageAsyncReservation(&stale)
								require.NoError(t, err)
								assert.True(t, refunded)
								resolved := assertReservation(0)
								assert.True(t, resolved.PrivateData.ImageAsync.BillingRefunded)
								return
							}
							stored := assertReservation(0)
							assert.JSONEq(t, `{"model_ratio":1}`, string(stored.PrivateData.ImageAsync.PriceDataJSON))
							assert.Zero(t, stored.PrivateData.ImageAsync.ImageQuotaBeforeGroup)
							assert.Empty(t, stored.PrivateData.ImageAsync.BillingRequestInputJSON)
							assert.Nil(t, stored.PrivateData.BillingContext)
						})
					}
				})
			}
			for _, source := range []string{"wallet", "subscription"} {
				for _, failTable := range []string{"", "tokens", "tasks"} {
					t.Run(source+"/fail_"+failTable, func(t *testing.T) {
						label := source + "_" + failTable
						user := User{Username: label, AffCode: label, Quota: 9000, UsedQuota: 700, RequestCount: 3}
						require.NoError(t, db.Create(&user).Error)
						token := Token{UserId: user.Id, Key: label, RemainQuota: 9000, UsedQuota: 1000}
						require.NoError(t, db.Create(&token).Error)
						channel := Channel{Name: label, UsedQuota: 700}
						require.NoError(t, db.Create(&channel).Error)
						sub := UserSubscription{UserId: user.Id, AmountTotal: 10000, AmountUsed: 1500}
						require.NoError(t, db.Create(&sub).Error)
						task := Task{TaskID: label, UserId: user.Id, ChannelId: channel.Id, Quota: 1000, Status: TaskStatusFailure, PrivateData: TaskPrivateData{BillingSource: source, SubscriptionId: sub.Id, TokenId: token.Id, ImageAsync: &ImageAsyncPrivateData{RequestHash: "preserved-hash"}}}
						require.NoError(t, db.Create(&task).Error)
						record := SubscriptionPreConsumeRecord{RequestId: "image_async:" + task.TaskID, UserId: user.Id, UserSubscriptionId: sub.Id, PreConsumed: 1000, Status: "consumed"}
						require.NoError(t, db.Create(&record).Error)
						stale := task
						if failTable != "" {
							const callback = "image-refund:fail-update"
							require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
								if tx.Statement.Table == failTable {
									_ = tx.AddError(errors.New("injected refund write failure"))
								}
							}))
							func() {
								defer func() { require.NoError(t, db.Callback().Update().Remove(callback)) }()
								won, err := RefundImageAsyncReservation(&task)
								require.ErrorContains(t, err, "injected refund write failure")
								assert.False(t, won)
							}()
							require.NoError(t, db.First(&user, user.Id).Error)
							require.NoError(t, db.First(&token, token.Id).Error)
							require.NoError(t, db.First(&sub, sub.Id).Error)
							require.NoError(t, db.First(&task, task.ID).Error)
							require.NoError(t, db.First(&record, record.Id).Error)
							assert.Equal(t, 9000, user.Quota)
							assert.Equal(t, 9000, token.RemainQuota)
							assert.Equal(t, 1000, token.UsedQuota)
							assert.EqualValues(t, 1500, sub.AmountUsed)
							assert.Equal(t, 1000, task.Quota)
							assert.False(t, task.PrivateData.ImageAsync.BillingRefunded)
							assert.Equal(t, "consumed", record.Status)
						}
						won, err := RefundImageAsyncReservation(&task)
						require.NoError(t, err)
						require.True(t, won)
						won, err = RefundImageAsyncReservation(&stale)
						require.NoError(t, err)
						assert.False(t, won, "a stale caller cannot refund twice")
						require.NoError(t, db.First(&user, user.Id).Error)
						require.NoError(t, db.First(&token, token.Id).Error)
						require.NoError(t, db.First(&channel, channel.Id).Error)
						require.NoError(t, db.First(&sub, sub.Id).Error)
						require.NoError(t, db.First(&record, record.Id).Error)
						wantWallet, wantSub, wantRecord := 10000, int64(1500), "consumed"
						if source == "subscription" {
							wantWallet, wantSub, wantRecord = 9000, 500, "refunded"
						}
						assert.Equal(t, wantWallet, user.Quota)
						assert.Equal(t, wantSub, sub.AmountUsed)
						assert.Equal(t, wantRecord, record.Status)
						assert.Equal(t, 10000, token.RemainQuota)
						assert.Zero(t, token.UsedQuota)
						assert.Equal(t, 700, user.UsedQuota)
						assert.Equal(t, 3, user.RequestCount)
						assert.EqualValues(t, 700, channel.UsedQuota)
						assert.Zero(t, stale.Quota)
						assert.True(t, stale.PrivateData.ImageAsync.BillingRefunded)
						assert.Equal(t, "preserved-hash", stale.PrivateData.ImageAsync.RequestHash)
					})
				}
			}
			t.Run("maintenance_after_completed_history", func(t *testing.T) {
				history := make([]Task, 101)
				for i := range history {
					history[i] = Task{TaskID: fmt.Sprintf("completed_%d", i), Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusSuccess, FinishTime: 1,
						PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{BillingFinalized: true, WebhookURL: "https://example.com/hook", WebhookDone: true}}}
				}
				require.NoError(t, db.Create(&history).Error)
				pending := Task{TaskID: "pending_after_history", Platform: constant.TaskPlatformSyncTask, Action: constant.ImageAsyncAction, Status: TaskStatusFailure, FinishTime: 1,
					PrivateData: TaskPrivateData{ImageAsync: &ImageAsyncPrivateData{WebhookURL: "https://example.com/hook", BodyPath: "pending.body"}}}
				require.NoError(t, db.Create(&pending).Error)
				for name, tasks := range map[string][]*Task{
					"billing": GetImageAsyncTerminalTasksNeedBilling(1),
					"webhook": GetImageAsyncTasksNeedWebhook(10, 1),
					"cleanup": GetTerminalImageAsyncTasksOlderThan(10, 1),
				} {
					t.Run(name, func(t *testing.T) {
						require.Len(t, tasks, 1)
						assert.Equal(t, pending.TaskID, tasks[0].TaskID)
					})
				}
			})
		})
	}
}

// Full startup uses unqualified legacy table names, so each configured external
// engine gets a newly created scratch database. Existing DSN tables are never
// dropped. The test account needs permission to create and drop databases.
func openMigrationStartupDatabase(t *testing.T, dialect string) (*gorm.DB, common.DatabaseType) {
	t.Helper()
	if dialect == "sqlite" {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "startup.db")), newGormConfig(true))
		require.NoError(t, err)
		return db, common.DatabaseTypeSQLite
	}
	dsnEnv := "TEST_MYSQL_DSN"
	dbType := common.DatabaseTypeMySQL
	if dialect == "postgres" {
		dsnEnv = "TEST_POSTGRES_DSN"
		dbType = common.DatabaseTypePostgreSQL
	}
	dsn := strings.TrimSpace(os.Getenv(dsnEnv))
	if dsn == "" {
		t.Skipf("%s is not configured", dsnEnv)
	}
	var adminDialector gorm.Dialector
	var fixtureDialector gorm.Dialector
	name := fmt.Sprintf("newapi_startup_%d", time.Now().UnixNano())
	if dialect == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		adminDialector = mysqlMigrationDialector{mysql.Dialector{Config: &mysql.Config{DSN: dsn}}}
		config.DBName = name
		fixtureDialector = mysqlMigrationDialector{mysql.Dialector{Config: &mysql.Config{DSN: config.FormatDSN()}}}
	} else {
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Contains(t, []string{"postgres", "postgresql"}, parsed.Scheme, "startup test requires a PostgreSQL URL DSN")
		adminDialector = postgresMigrationDialector{postgres.Dialector{Config: &postgres.Config{DSN: dsn, PreferSimpleProtocol: true}}}
		parsed.Path = "/" + name
		fixtureDialector = postgresMigrationDialector{postgres.Dialector{Config: &postgres.Config{DSN: parsed.String(), PreferSimpleProtocol: true}}}
	}
	admin, err := gorm.Open(adminDialector, newGormConfig(false))
	require.NoError(t, err)
	adminSQL, err := admin.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, adminSQL.Close()) })
	require.NoError(t, admin.Exec("CREATE DATABASE ?", clause.Table{Name: name}).Error)
	t.Cleanup(func() { require.NoError(t, admin.Exec("DROP DATABASE ?", clause.Table{Name: name}).Error) })
	db, err := gorm.Open(fixtureDialector, newGormConfig(dialect != "postgres"))
	require.NoError(t, err)
	return db, dbType
}

func mustColumnTypes(t *testing.T, db *gorm.DB, model any) []gorm.ColumnType {
	t.Helper()
	columns, err := db.Migrator().ColumnTypes(model)
	require.NoError(t, err)
	return columns
}

func dropMigrationStartupTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	models := []any{
		&Channel{}, &Token{}, &User{}, &UserSession{}, &AuthFlow{}, &ExternalIdentityClaim{},
		&PasskeyCredential{}, &Option{}, &LoginEncryptionKey{}, &Redemption{}, &Ability{}, &Log{},
		&Midjourney{}, &TopUp{}, &QuotaData{}, &Task{}, &ImageAsyncIdempotency{}, &TaskPlugin{},
		&Model{}, &Vendor{}, &PrefillGroup{}, &Setup{}, &TwoFA{}, &TwoFABackupCode{}, &Checkin{},
		&SubscriptionOrder{}, &UserSubscription{}, &SubscriptionPreConsumeRecord{}, &CustomOAuthProvider{},
		&UserOAuthBinding{}, &PerfMetric{}, &SystemInstance{}, &SystemTask{}, &SystemTaskLock{},
		&CasbinRule{}, &AuthzRole{}, &UserAccessToken{}, &SubscriptionPlan{},
	}
	if err := db.Migrator().DropTable(models...); err != nil {
		t.Fatalf("drop startup fixture tables: %v", err)
	}
}
