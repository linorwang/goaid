# SMS Service Integration Implementation Outline

## 椤圭洰姒傝堪
鍦?sendsms 鐩綍涓嬪疄鐜板鐭俊鏈嶅姟鍟嗛泦鎴愭柟妗堬紝鏀寔 Redis 缂撳瓨锛堟帴鏀?redis.Client 鎴?redis.Cmdable 浣滀负鍙傛暟锛夛紝鎻愪緵缁熶竴鐨勭煭淇″彂閫佹帴鍙ｃ€傚叿澶囧畬鍠勭殑瀹归敊鏈哄埗鍜?Failover 鏀寔锛岀‘淇濋珮鍙敤鎬с€?

## 鐩綍缁撴瀯璁捐

```
sendsms/
鈹溾攢鈹€ README.md                    # 浣跨敤鏂囨。
鈹溾攢鈹€ types.go                     # 鏍稿績绫诲瀷瀹氫箟
鈹溾攢鈹€ interface.go                 # 缁熶竴鎺ュ彛瀹氫箟
鈹溾攢鈹€ factory.go                   # 鏈嶅姟宸ュ巶
鈹溾攢鈹€ cache.go                     # Redis 缂撳瓨瀹炵幇
鈹溾攢鈹€ retry.go                     # 閲嶈瘯鍜屽閿欐満鍒?
鈹溾攢鈹€ failover.go                  # Failover 绛栫暐瀹炵幇
鈹溾攢鈹€ provider/                    # 鏈嶅姟鍟嗗疄鐜?
鈹?  鈹溾攢鈹€ aliyun.go                # 闃块噷浜?
鈹?  鈹溾攢鈹€ tencent.go               # 鑵捐浜?
鈹?  鈹溾攢鈹€ baidu.go                 # 鐧惧害浜?
鈹?  鈹溾攢鈹€ huawei.go                # 鍗庝负浜?
鈹?  鈹溾攢鈹€ netease.go               # 缃戞槗浜戜俊
鈹?  鈹溾攢鈹€ ronglian.go              # 瀹硅仈浜?
鈹?  鈹溾攢鈹€ aurora.go                # 鏋佸厜
鈹?  鈹溾攢鈹€ chuanglan.go             # 鍒涜摑253
鈹?  鈹斺攢鈹€ twilio.go                # Twilio (鍥介檯)
鈹溾攢鈹€ sms.go                       # SMS 鏍稿績瀹㈡埛绔?
鈹溾攢鈹€ config.go                    # 閰嶇疆绠＄悊
鈹溾攢鈹€ errors.go                    # 閿欒瀹氫箟
鈹斺攢鈹€ example.go                   # 浣跨敤绀轰緥
```

## 鏍稿績缁勪欢璁捐

### 1. types.go - 鏍稿績绫诲瀷瀹氫箟

```go
// 鐭俊绫诲瀷
type SMSType int
const (
    SMSVerification SMSType = iota  // 楠岃瘉鐮?
    SMSNotification                // 閫氱煡绫?
    SMSMarketing                   // 钀ラ攢绫?
)

// 鐭俊璇锋眰
type SMSRequest struct {
    Phone     string   // 鎵嬫満鍙?
    Template  string   // 妯℃澘ID
    Content   string   // 鐭俊鍐呭锛堥儴鍒嗘湇鍔″晢锛?
    Params    []string // 妯℃澘鍙傛暟
    SignName  string   // // 绛惧悕
    Type      SMSType  // 鐭俊绫诲瀷
    ExtID     string   // 鎵╁睍ID锛堢敤浜庡洖璋冨尮閰嶏級
    RetryCount int    // 宸查噸璇曟鏁帮紙鍐呴儴浣跨敤锛?
}

// 鐭俊鍝嶅簲
type SMSResponse struct {
    Success     bool   // 鏄惁鎴愬姛
    MessageID   string // 娑堟伅ID
    Message     string // 杩斿洖娑堟伅
    Code        string // 鐘舵€佺爜
    Cost        float64// 璐圭敤
    RequestID   string // 璇锋眰ID
    Provider    string // 鏈嶅姟鍟嗗悕绉?
    RetryCount  int    // 瀹為檯閲嶈瘯娆℃暟
    Duration      time.Duration // 鑰楁椂
    Error      error  // 閿欒淇℃伅
}

// 楠岃瘉鐮佽姹?
type VerificationCodeRequest struct {
    Phone      string
    ExpireTime time.Duration // 杩囨湡鏃堕棿
    CodeLength int          // 楠岃瘉鐮侀暱搴?
    Template   string       // 妯℃澘ID
    SignName   string       // 绛惧悕
}

// 楠岃瘉鐮侀獙璇佽姹?
type VerifyCodeRequest struct {
    Phone     string
    Code      string
    CleanOnce bool // 楠岃瘉鍚庢槸鍚﹀垹闄?
}

// 楠岃瘉鐮佸搷搴?
type VerifyResult struct {
    Valid   bool
    Message string
}

// 缂撳瓨閰嶇疆
type CacheConfig struct {
    Prefix           string        // Redis key 鍓嶇紑
    ExpireTime       time.Duration // 榛樿杩囨湡鏃堕棿
    VerificationExp  time.Duration // 楠岃瘉鐮佽繃鏈熸椂闂?
    EnableLimit      bool          // 鏄惁鍚敤闄愭祦
    LimitCount       int           // 闄愭祦娆℃暟
    LimitWindow      time.Duration // 闄愭祦鏃堕棿绐楀彛
}

// 鎵归噺鍙戦€佺粨鏋?
type BatchResult struct {
    Total      int             // 鎬绘暟
    Success    int             // 鎴愬姛鏁?
    Failed     int             // 澶辫触鏁?
    Responses  []*SMSResponse  // 鎵€鏈夊搷搴?
    FailedReqs []*SMSRequest   // 澶辫触鐨勮姹傦紙鍙噸璇曪級
}

// 鏈嶅姟鍟嗗仴搴风姸鎬?
type ProviderHealth struct {
    Name           string
    IsHealthy      bool
    ErrorCount     int
    LastErrorTime  time.Time
    LastCheckTime  time.Time
    FailoverCount  int
}

// 閲嶈瘯绛栫暐绫诲瀷
type RetryStrategy int
const (
    RetryFixedDelay RetryStrategy = iota // 鍥哄畾寤惰繜
    RetryExponentialBackoff             // 鎸囨暟閫€閬?
    RetryLinearBackoff                  // 绾挎€ч€€閬?
)

// Failover 绛栫暐绫诲瀷
type FailoverStrategy int
const (
    FailoverSequential FailoverStrategy = iota // 椤哄簭鍒囨崲
    FailoverRandom                           // 闅忔満閫夋嫨
    FailoverRoundRobin                       // 杞
)

// 閿欒绫诲瀷
type ErrorType int
const (
    ErrorTypeNetwork ErrorType = iota // 缃戠粶閿欒
    ErrorTypeTimeout                  // 瓒呮椂閿欒
    ErrorTypeProvider                // 鏈嶅姟鍟嗛敊璇?
    ErrorTypeRateLimit               // 闄愭祦閿欒
    ErrorTypeAuth                    // 璁よ瘉閿欒
    ErrorTypeInvalid                 // 璇锋眰鏃犳晥
)
```

### 2. interface.go - 缁熶竴鎺ュ彛瀹氫箟

```go
// SMSProvider 鏈嶅姟鍟嗘帴鍙?
type SMSProvider interface {
    // 鍙戦€佺煭淇★紙甯﹀閿欏拰閲嶈瘯锛?
    Send(req *SMSRequest) (*SMSResponse, error)
    
    // 鎵归噺鍙戦€侊紙甯﹀閿欙級
    SendBatch(reqs []*SMSRequest) ([]*SMSResponse, error)
    
    // 鑾峰彇鏈嶅姟鍟嗗悕绉?
    Name() string
    
    // 妫€鏌ラ厤缃槸鍚︽湁鏁?
    ValidateConfig() error
    
    // 鑾峰彇浣欓锛堝鏋滄敮鎸侊級
    GetBalance() (*Balance, error)
    
    // 鍋ュ悍妫€鏌?
    HealthCheck() bool
    
    // 鑾峰彇閿欒绫诲瀷
    GetErrorType(err error) ErrorType
    
    // 鍒ゆ柇閿欒鏄惁鍙噸璇?
    IsRetryable(err error) bool
}

// 楠岃瘉鐮佺鐞嗘帴鍙?
type VerificationManager interface {
    // 鍙戦€侀獙璇佺爜
    SendCode(req *VerificationCodeRequest) (*SMSResponse, error)
    
    // 楠岃瘉楠岃瘉鐮?
    VerifyCode(req *VerifyCodeRequest) (*VerifyResult, error)
    
    // 妫€鏌ユ槸鍚﹀彲浠ュ彂閫侊紙闄愭祦妫€鏌ワ級
    CanSend(phone) (bool, error)
}

// FailoverManager Failover 绠＄悊鎺ュ彛
type FailoverManager interface {
    // 鑾峰彇鍙敤鏈嶅姟鍟?
    GetAvailableProvider() SMSProvider
    
    // 鏍囪鏈嶅姟鍟嗗け璐?
    MarkProviderFailed(provider string)
    
    // 鏍囪鏈嶅姟鍟嗘仮澶?
    MarkProviderHealthy(provider string)
    
    // 鑾峰彇鎵€鏈夋湇鍔″晢鍋ュ悍鐘舵€?
    GetHealthStatus() []*ProviderHealth
}
```

### 3. config.go - 閰嶇疆绠＄悊

```go
// ProviderConfig 鏈嶅姟鍟嗛厤缃?
type ProviderConfig struct {
    Aliyun   *AliyunConfig
    Tencent  *TencentConfig
    Baidu    *BaiduConfig
    Huawei   *HuaweiConfig
    Netease  *NeteaseConfig
    Ronglian *RonglianConfig
    Aurora   *AuroraConfig
    Chuanglan *ChuanglanConfig
    Twilio   *TwilioConfig
}

// Config 鎬婚厤缃?
type Config struct {
    // 鏈嶅姟鍟嗛厤缃?
    PrimaryProvider   string   // 涓绘湇鍔″晢
    BackupProviders   []string // 澶囩敤鏈嶅姟鍟嗗垪琛?
    
    // 榛樿璁剧疆
    DefaultSign      string   // 榛樿绛惧悕
    DefaultTemplate  string   // 榛樿妯℃澘ID
    
    // 缂撳瓨閰嶇疆
    CacheConfig      *CacheConfig
    
    // 閲嶈瘯閰嶇疆
    RetryStrategy    RetryStrategy // 閲嶈瘯绛栫暐
    RetryTimes       int            // 閲嶈瘯娆℃暟
    RetryDelay       time.Duration  // 鍒濆閲嶈瘯寤惰繜
    MaxRetryDelay    time.Duration  // 鏈€澶ч噸璇曞欢杩?
    RetryMultiplier  float64        // 閫€閬夸箻鏁帮紙鎸囨暟/绾挎€э級
    
    // Failover 閰嶇疆
    EnableFailover   bool               // 鏄惁鍚敤 Failover
    FailoverStrategy FailoverStrategy   // Failover 绛栫暐
    FailoverCooldown  time.Duration      // Failover 鍐峰嵈鏃堕棿
    HealthCheckInterval time.Duration    // 鍋ュ悍妫€鏌ラ棿闅?
    
    // 璇锋眰閰嶇疆
    Timeout         time.Duration  // 璇锋眰瓒呮椂
    BatchSize       int            // 鎵归噺鍙戦€佸ぇ灏?
    ConcurrentLimit int            // 骞跺彂闄愬埗
    
    // 瀹归敊閰嶇疆
    EnableCircuitBreaker bool           // 鏄惁鍚敤鐔旀柇鍣?
    CircuitBreakerThreshold int         // 鐔旀柇闃堝€?
    CircuitBreakerTimeout   time.Duration // 鐔旀柇瓒呮椂
}

// 鍏蜂綋鏈嶅姟鍟嗘湇鍔￠厤缃紙浠ラ樋閲屼簯涓轰緥锛?
type AliyunConfig struct {
    AccessKeyID     string
    AccessKeySecret string
    RegionID        string // 榛樿 cn-hangzhou
    SignName        string
    Endpoint        string // 榛樿 dysmsapi.aliyuncs.com
}

// 鍏朵粬鏈嶅姟鍟嗛厤缃被浼?..
```

### 4. retry.go - 閲嶈瘯鍜屽閿欐満鍒?

```go
// RetryManager 閲嶈瘯绠＄悊鍣?
type RetryManager struct {
    config *Config
}

// NewRetryManager 鍒涘缓閲嶈瘯绠＄悊鍣?
func NewRetryManager(config *Config) *RetryManager

// Retry 甯﹂噸璇曠殑鍙戦€?
func (r *RetryManager) Retry(ctx context.Context, fn func() (*SMSResponse, error)) (*SMSResponse, error)

// GetDelay 鑾峰彇閲嶈瘯寤惰繜
func (r *RetryManager) GetDelay(attempt int) time.Duration

// ShouldRetry 鍒ゆ柇鏄惁搴旇閲嶈瘯
func (r *RetryManager) ShouldRetry(err error, attempt int) bool

// SleepWithCancel 鍙彇娑堢殑鐫＄湢
func (r *RetryManager) SleepWithCancel(ctx context.Context, delay time.Duration) error
```

### 5. failover.go - Failover 绛栫暐瀹炵幇

```go
// FailoverManager Failover 绠＄悊鍣?
type FailoverManager struct {
    providers      map[string]SMSProvider
    primary        string
    backups        []string
    strategy       FailoverStrategy
    healthStatus   map[string]*ProviderHealth
    cooldown       time.Duration
    mu             sync.RWMutex
    currentIndex   int // 杞绱㈠紩
}

// NewFailoverManager 鍒涘缓 Failover 绠＄悊鍣?
func NewFailoverManager(primary string, backups []string, providers map[string]SMSProvider, strategy FailoverStrategy, cooldown time.Duration) *FailoverManager

// GetAvailableProvider 鑾峰彇鍙敤鏈嶅姟鍟?
func (f *FailoverManager) GetAvailableProvider() SMSProvider

// MarkProviderFailed 鏍囪鏈嶅姟鍟嗗け璐?
func (f *FailoverManager) MarkProviderFailed(provider string)

// MarkProviderHealthy 鏍囪鏈嶅姟鍟嗘仮澶?
func (f *FailoverManager) MarkProviderHealthy(provider string)

// GetHealthStatus 鑾峰彇鍋ュ悍鐘舵€?
func (f *FailoverManager) GetHealthStatus() []*ProviderHealth

// isInCooldown 鍒ゆ柇鏄惁鍦ㄥ喎鍗存湡
func (f *FailoverManager) isInCooldown(provider string) bool

// getSequentialProvider 椤哄簭鑾峰彇鏈嶅姟鍟?
func (f *FailoverManager) getSequentialProvider() SMSProvider

// getRandomProvider 闅忔満鑾峰彇鏈嶅姟鍟?
func (f *FailoverManager) getRandomProvider() SMSProvider

// getRoundRobinProvider 杞鑾峰彇鏈嶅姟鍟?
func (f *FailoverManager) getRoundRobinProvider() SMSProvider
```

### 6. cache.go - Redis 缂撳瓨瀹炵幇

```go
// SMSCache Redis 缂撳瓨绠＄悊
type SMSCache struct {
    client   redis.Cmdable // 鎺ユ敹 redis.Client 鎴?redis.Cmdable
    config   *CacheConfig
}

// NewSMSCache 鍒涘缓缂撳瓨瀹炰緥
func NewSMSCache(client redis.Cmdable, config *CacheConfig) *SMSCache

// SaveCode 淇濆瓨楠岃瘉鐮?
func (c *SMSCache) SaveCode(phone, code string, expire time.Duration) error

// GetCode 鑾峰彇楠岃瘉鐮?
func (c *SMSCache) GetCode(phone string) (string, error)

// DeleteCode 鍒犻櫎楠岃瘉鐮?
func (c *SMSCache) DeleteCode(phone string) error

// CheckLimit 妫€鏌ュ彂閫侀鐜囬檺鍒?
func (c *SMSCache) CheckLimit(phone string) (bool, error)

// RecordAttempt 璁板綍鍙戦€佸皾璇?
func (c *SMSCache) RecordAttempt(phone string) error

// SaveFailoverRecord 淇濆瓨 Failover 璁板綍
func (c *SMSCache) SaveFailoverRecord(phone, failedProvider, successProvider string) error

// GetFailoverRecords 鑾峰彇 Failover 璁板綍
func (c *SMSCache) GetFailoverRecords(phone string, limit int) ([]string, error)

// GetKey 鐢熸垚Redis key
func (c *SMSCache) GetKey(suffix string) string
```

### 7. sms.go - SMS 鏍稿績瀹㈡埛绔紙澧炲己鐗堬級

```go
// SMSClient 鐭俊瀹㈡埛绔?
type SMSClient struct {
    provider      SMSProvider
    failoverMgr   *FailoverManager
    retryMgr      *RetryManager
    cache         *SMSCache
    config        *Config
}

// NewSMSClient 鍒涘缓鐭俊瀹㈡埛绔?
func NewSMSClient(primary string, backups []string, providers map[string]SMSProvider, cache redis.Cmdable, config *Config) (*SMSClient, error)

// Send 鍙戦€佺煭淇★紙甯﹀閿欏拰 Failover锛?
func (c *SMSClient) Send(req *SMSRequest) (*SMSResponse, error)

// SendWithRetry 甯﹂噸璇曠殑鍙戦€?
func (c *SMSClient) SendWithRetry(provider SMSProvider, req *SMSRequest) (*SMSResponse, error)

// SendBatch 鎵归噺鍙戦€侊紙甯﹀閿欙級
func (c *SMSClient) SendBatch(reqs []*SMSRequest) (*BatchResult, error)

// SendBatchConcurrent 骞跺彂鎵归噺鍙戦€?
func (c *SMSClient) SendBatchConcurrent(reqs []*SMSRequest, concurrency int) (*BatchResult, error)

// SendVerificationCode 鍙戦€侀獙璇佺爜
func (c *SMSClient) SendVerificationCode(req *VerificationCodeRequest) (*SMSResponse, error)

// VerifyCode 楠岃瘉楠岃瘉鐮?
func (c *SMSClient) VerifyCode(req *VerifyCodeRequest) (*VerifyResult, error)

// SendWithTemplate 浣跨敤妯℃澘鍙戦€?
func (c *SMSClient) SendWithTemplate(phone, templateID, signName string, params []string) (*SMSResponse, error)

// GetHealthStatus 鑾峰彇鎵€鏈夋湇鍔″晢鍋ュ悍鐘舵€?
func (c *SMSClient) GetHealthStatus() []*ProviderHealth

// RetryFailed 閲嶈瘯澶辫触鐨勬壒閲忓彂閫?
func (c *SMSClient) RetryFailed(failedReqs []*SMSRequest) (*BatchResult, error)
```

### 8. factory.go - 鏈嶅姟宸ュ巶

```go
// NewProvider 鏍规嵁閰嶇疆鍒涘缓鏈嶅姟鍟嗗疄渚?
func NewProvider(providerType string, config *ProviderConfig) (SMSProvider, error)

// NewProviders 鎵归噺鍒涘缓鏈嶅姟鍟嗗疄渚?
func NewProviders(providerNames []string, config *ProviderConfig) (map[string]SMSProvider, error)

// 鏀寔鐨勬湇鍔″晢鍒楄〃
const (
    ProviderAliyun   = "aliyun"
    ProviderTencent  = "tencent"
    ProviderBaidu    = "baidu"
    ProviderHuawei   = "huawei"
    ProviderNetease  = "netease"
    ProviderRonglian = "ronglian"
    ProviderAurora   = "aurora"
    ProviderChuanglan= "chuanglan"
    ProviderTwilio   = "twilio"
)
```

### 9. provider/aliyun.go - 闃块噷浜戝疄鐜扮ず渚?

```go
type AliyunProvider struct {
    config *AliyunConfig
    client *dysmsapi.Client
}

func NewAliyunProvider(config *AliyunConfig) (*AliyunProvider, error)

func (p *AliyunProvider) Send(req *SMSRequest) (*SMSResponse, error)

func (p *AliyunProvider) SendBatch(reqs []*SMSRequest) ([]*SMSResponse, error)

func (p *AliyunProvider) Name() string

func (p *AliyunProvider) ValidateConfig() error

func (p *AliyunProvider) GetBalance() (*Balance, error)

func (p *AliyunProvider) HealthCheck() bool

func (p *AliyunProvider) GetErrorType(err error) ErrorType

func (p *AliyunProvider) IsRetryable(err error) bool
```

### 10. errors.go - 閿欒瀹氫箟

```go
var (
    ErrInvalidPhone      = errors.New("invalid phone number")
    ErrEmptyTemplate     = errors.New("template ID is empty")
    ErrEmptySignName     = errors.New("sign name is empty")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
    ErrCodeNotFound      = errors.New("verification code not found")
    ErrCodeExpired       = errors.New("verification code expired")
    ErrCodeInvalid       = errors.New("verification code invalid")
    ErrSendFailed        = errors.New("send SMS failed")
    ErrConfigInvalid     = errors.New("invalid configuration")
    ErrAllProvidersFailed = errors.New("all providers failed")
    ErrTimeout           = errors.New("request timeout")
    ErrNetworkError      = errors.New("network error")
    ErrAuthFailed        = errors.New("authentication failed")
)

// SMSError 鐭俊閿欒鍖呰
type SMSError struct {
    Code       string
    Message    string
    ErrorType  ErrorType
    Retryable  bool
    Provider   string
    Original   error
}

func (e *SMSError) Error() string
func (e *SMSError) Unwrap() error
```

## 鏍稿績鍔熻兘鐗规€?

### 1. 缁熶竴鎺ュ彛
- 鎵€鏈夋湇鍔″晢瀹炵幇缁熶竴鐨?SMSProvider 鎺ュ彛
- 鏀寔鍔ㄦ€佸垏鎹㈡湇鍔″晢
- 缁熶竴鐨勮姹?鍝嶅簲鏍煎紡

### 2. Redis 缂撳瓨鍔熻兘
- **楠岃瘉鐮佺鐞?*
  - 鐢熸垚骞跺瓨鍌ㄩ獙璇佺爜
  - 楠岃瘉楠岃瘉鐮?
  - 璁剧疆杩囨湡鏃堕棿
  
- **棰戠巼闄愬埗**
  - 闃叉鎭舵剰鍒锋帴鍙?
  - 鍙厤缃檺娴佺瓥鐣?
  - 浣跨敤 Redis 璁℃暟鍣ㄥ疄鐜?
  
- **鍙戦€佽褰?*
  - 璁板綍鍙戦€?
鍘嗗彶
  - 渚夸簬闂杩借釜
  - 鏀寔缁熻鍒嗘瀽
  
- **Failover 璁板綍**
  - 璁板綍鍒囨崲鍘嗗彶
  - 缁熻鍒囨崲娆℃暟
  - 渚夸簬闂鎺掓煡

### 3. 楠岃瘉鐮佸姛鑳?
- 鑷姩鐢熸垚闅忔満楠岃瘉鐮?
- 鏀寔鑷畾涔夐暱搴︼紙4-8浣嶏級
- 鑷姩杩囨湡鏈哄埗
- 楠岃瘉鍚庡彲閫夋嫨鍒犻櫎
- 闆嗘垚棰戠巼闄愬埗

### 4. 瀹归敊鍜?Failover 鏈哄埗
- **鑷姩閲嶈瘯**
  - 鍙厤缃噸璇曟鏁板拰寤惰繜
  - 鎸囨暟閫€閬跨瓥鐣?
  - 浠呭鍙噸璇曢敊璇噸璇?
  - 鏀寔鍙栨秷閲嶈瘯
  
- **澶氭湇鍔″晢 Failover**
  - 涓绘湇鍔″晢澶辫触鑷姩鍒囨崲澶囩敤鏈嶅姟鍟?
  - 鏀寔閰嶇疆澶氫釜澶囩敤鏈嶅姟鍟?
  - 鑷姩鍋ュ悍妫€鏌?
  - 鏀寔澶氱 Failover 绛栫暐锛?
    - Sequential: 椤哄簭鍒囨崲锛堥粯璁わ級
    - Random: 闅忔満閫夋嫨
    - RoundRobin: 杞閫夋嫨
  - Failover 鍐峰嵈鏈哄埗锛堥伩鍏嶉绻佸垏鎹級
  
- **鍗曚釜鍙戦€佸閿?*
  - 鍙戦€佸け璐ヨ嚜鍔ㄩ噸璇?
  - 涓绘湇鍔″晢澶辫触鍒囨崲澶囩敤
  - 璇︾粏鐨勫け璐ュ師鍥犺褰?
  - 璁板綍閲嶈瘯娆℃暟鍜岃€楁椂
  
- **鎵归噺鍙戦€佸閿?*
  - 鍗曚釜澶辫触涓嶅奖鍝嶅叾浠?
  - 鏀寔閮ㄥ垎鎴愬姛杩斿洖
  - 澶辫触椤规敮鎸侀噸鏂板彂閫?
  - 鎵归噺浠诲姟闃熷垪绠＄悊
  - 鏀寔骞跺彂鎺у埗
  
- **閿欒闅旂**
  - 鏈嶅姟鍟嗛敊璇殧绂?
  - 缃戠粶閿欒闅旂
  - 涓氬姟閿欒闅旂
  
- **鐔旀柇鍣ㄦ満鍒讹紙鍙€夛級**
  - 杈惧埌闃堝€艰嚜鍔ㄧ啍鏂?
  - 鐔旀柇鍚庤嚜鍔ㄦ仮澶?
  - 闃叉闆穿鏁堝簲

### 5. 閲嶈瘯绛栫暐
- **鍥哄畾寤惰繜閲嶈瘯**
  - 姣忔閲嶈瘯闂撮殧鐩稿悓
  - 閫傚悎绋冲畾鐨勭幆澧?
  
- **鎸囨暟閫€閬块噸璇?*
  - 寤惰繜鏃堕棿鎸囨暟澧為暱
  - 閫傚悎缃戠粶涓嶇ǔ瀹氱幆澧?
  - 鍏紡: delay = min(initialDelay * (2^attempt), maxDelay)
  
-**绾挎€ч€€閬块噸璇?*
  - 寤惰繜鏃堕棿绾挎€у闀?
  - 閫傚悎鍙帶鐨勫け璐ュ満鏅?
  - 鍏紡: delay = min(initialDelay + (multiplier * attempt), maxDelay)

### 6. 鍋ュ悍妫€鏌?
- 瀹氭湡鍋ュ悍妫€鏌?
- 妫€鏌ユ湇鍔″晢鍙敤鎬?
- 鑷姩鏍囪鍋ュ悍/涓嶅仴搴风姸鎬?
- 鏀寔鎵嬪姩瑙﹀彂妫€鏌?

### 7. 閿欒澶勭悊
-缁熶竴鐨勯敊璇畾涔?
- 璇︾粏鐨勯敊璇俊鎭?
- 閿欒绫诲瀷鍒嗙被
- 鏀寔閿欒鍖呰
- 閿欒鍙噸璇曟€у垽鏂?

### 8. 閰嶇疆绠＄悊
- 鏀寔澶氭湇鍔″晢閰嶇疆
- 榛樿閰嶇疆鏀寔
- 閰嶇疆楠岃瘉
- 鐜鍙橀噺鏀寔
- 鐑噸杞介厤缃紙鍙€夛級

### 9. 鎵╁睍鎬?
- 鏄撲簬娣诲姞鏂版湇鍔″晢
- 鎻掍欢鍖栬璁?
- 閽╁瓙鍑芥暟鏀寔
- 涓棿浠舵敮鎸?

## 瀹归敊鍜?Failover 璇︾粏瀹炵幇

### 鍗曚釜鐭俊鍙戦€佸閿欐祦绋?

```
1. 璇锋眰寮€濮?
   鈫?
2. 闄愭祦妫€鏌?
   鈹溾攢 閫氳繃 鈫?缁х画
   鈹斺攢 澶辫触 鈫?杩斿洖闄愭祦閿欒
   鈫?
3. 鑾峰彇鏈嶅姟鍟嗭紙鑰冭檻 Failover锛?
   鈹溾攢 涓绘湇鍔″晢鍙敤 鈫?浣跨敤涓绘湇鍔″晢
   鈹斺攢 涓绘湇鍔″晢涓嶅彲鐢?鈫?灏濊瘯澶囩敤鏈嶅姟鍟?
   鈫?
4. 鍙戦€佺煭淇?
   鈹溾攢 鎴愬姛 鈫?璁板綍鎴愬姛锛岃繑鍥炵粨鏋?
   鈹斺攢 澶辫触 鈫?缁х画涓嬩竴姝?
   鈫?
5. 鍒ゆ柇閿欒绫诲瀷
   鈹溾攢 鍙噸璇曢敊璇?鈫?杩涘叆閲嶈瘯閫昏緫
   鈹斺攢 涓嶅彲閲嶈瘯閿欒 鈫?瑙﹀彂 Failover
   鈫?
6. 閲嶈瘯閫昏緫
   鈹溾攢 鏈揪鍒伴噸璇曟鏁?鈫?寤惰繜鍚庨噸璇?
   鈹斺攢 杈惧埌閲嶈瘯娆℃暟 鈫?瑙﹀彂 Failover
   鈫?
7. Failover 閫昏緫
   鈹溾攢 鏍囪褰撳墠鏈嶅姟鍟嗗け璐?
   鈹溾攢 鍒囨崲鍒板鐢ㄦ湇鍔″晢
   鈹溾攢 浣跨敤澶囩敤鏈嶅姟鍟嗛噸鏂板彂閫?
   鈹斺攢 鎵€鏈夋湇鍔″晢閮藉け璐?鈫?杩斿洖澶辫触
   鈫?
8. 杩斿洖缁撴灉
```

### 鎵归噺鍙戦€佸閿欐祦绋?

```
1. 鎺ユ敹鎵归噺璇锋眰
   鈫?
2. 鎵归噺澶у皬妫€鏌?
   鈹溾攢 鏈秴杩囬檺鍒?鈫?缁х画
   鈹斺攢 瓒呰繃闄愬埗 鈫?鍒嗘壒澶勭悊
   鈫?
3. 骞跺彂鎺у埗
   鈹溾攢 鏍规嵁閰嶇疆闄愬埗骞跺彂鏁?
   鈹斺攢 浣跨敤 worker pool 鎴?semaphore
   鈫?
4. 閫愪釜鍙戦€侊紙骞跺彂锛?
   姣忎釜璇锋眰锛?
   鈹溾攢 鍗曠嫭鐨勫閿欏鐞?
   鈹溾攢 澶辫触涓嶅奖鍝嶅叾浠?
   鈹斺攢 璁板綍鎴愬姛/澶辫触
   鈫?
5. 姹囨€荤粨鏋?
   鈹溾攢 缁熻鎴愬姛鏁?
   鈹溾攢 缁熻澶辫触鏁?
   鈹斺攢 鏀堕泦澶辫触鐨勮姹?
   鈫?
6. 杩斿洖鎵归噺缁撴灉
   鈹溾攢 鍖呭惈鎵€鏈夊搷搴?
   鈹斺攢 鍙噸璇曞け璐ョ殑璇锋眰
```

### Failover 绛栫暐瀹炵幇

#### 1. Sequential锛堥『搴忓垏鎹級
```
涓绘湇鍔″晢 鈫?澶囩敤1 鈫?澶囩敤2 鈫?澶囩敤3 鈫?... 鈫?鍏ㄩ儴澶辫触
```
- 浼樼偣: 绠€鍗曞彲闈狅紝浼樺厛浣跨敤涓绘湇鍔″晢
- 缂虹偣: 濡傛灉涓绘湇鍔″晢缁忓父澶辫触锛屼細鏈夊欢杩?
- 閫傜敤: 涓绘湇鍔″晢绋冲畾锛屽鐢ㄧ敤浜庡簲鎬?

#### 2. Random锛堥殢鏈洪€夋嫨锛?
```
闅忔満閫夋嫨鍋ュ悍鐨勬湇鍔″晢鍙戦€?
```
- 浼樼偣: 璐熻浇鍧囪　锛屽垎鏁ｉ闄?
- 缂虹偣: 鍙兘浼氶绻佸垏鎹?
- 閫傜敤: 澶氫釜鏈嶅姟鍟嗚兘鍔涚浉褰?

#### 3. RoundRobin锛堣疆璇級
```
鏈嶅姟鍟? 鈫?鏈嶅姟鍟? 鈫?鏈嶅姟鍟? 鈫?鏈嶅姟鍟? 鈫?...
```
- 浼樼偣: 璐熻浇鍧囪　锛屽彲棰勬祴
- 缂虹偣: 鍙兘浼氫娇鐢ㄤ笉鍋ュ悍鐨勬湇鍔″晢
- 閫傜敤: 闇€瑕佸钩鍧囧垎閰嶈礋杞?

### 閲嶈瘯绛栫暐瀹炵幇

#### 1. Fixed Delay锛堝浐瀹氬欢杩燂級
```
Retry 1: delay = 1s
Retry 2: delay = 1s
Retry 3: delay = 1s
...
```

#### 2. Exponential Backoff锛堟寚鏁伴€€閬匡級
```
Retry 1: delay = 1s
Retry 2: delay = 2s
Retry 3: delay = 4s
Retry 4: delay = 8s (max 10s)
Retry 5: delay = 10s
...
```

#### 3. Linear Backoff锛堢嚎鎬ч€€閬匡級
```
Retry 1: delay = 1s
Retry 2: delay = 2s
Retry 3: delay = 3s
Retry 4: delay = 4s
...
```

### 閿欒鍒嗙被鍜屽彲閲嶈瘯鎬у垽鏂?

```go
// 缃戠粶閿欒 - 鍙噸璇?
ErrorTypeNetwork 鈫?Retryable: true

// 瓒呮椂閿欒 - 鍙噸璇?
ErrorTypeTimeout 鈫?Retryable: true

// 闄愭祦閿欒 - 鍙噸璇曪紙闇€寤惰繜锛?
ErrorTypeRateLimit 鈫?Retryable: true

// 璁よ瘉閿欒 - 涓嶅彲閲嶈瘯
ErrorTypeAuth 鈫?Retryable: false

// 璇锋眰鏃犳晥 - 涓嶅彲閲嶈瘯
ErrorTypeInvalid 鈫?Retryable: false

// 鏈嶅姟鍟嗛敊璇?- 鏍规嵁鍏蜂綋閿欒鐮佸垽鏂?
ErrorTypeProvider 鈫?鏍规嵁閿欒鐮佸垽鏂?
```

## 浣跨敤绀轰緥

### 鍩虹浣跨敤绀轰緥

```go
package main

import (
    "context"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/linorwang/goaid/sendsms"
)

func main() {
    // 1. 鍒涘缓 Redis 瀹㈡埛绔?
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 2. 閰嶇疆鏈嶅姟鍟?
    providerConfig := &sendsms.ProviderConfig{
        Aliyun: &sendsms.AliyunConfig{
            AccessKeyID:     "your_access_key_id",
            AccessKeySecret: "your_access_key_secret",
            RegionID:        "cn-hangzhou",
            SignName:        "your_sign_name",
        },
        Tencent: &sendsms.TencentConfig{
            SecretID:  "your_secret_id",
            SecretKey: "your_secret_key",
            Region:    "ap-guangzhou",
            SignName:  "your_sign_name",
        },
    }
    
    // 3. 鍒涘缓鏈嶅姟鍟嗗疄渚?
    providers, err := sendsms.NewProviders(
        []string{"aliyun", "tencent"},
        providerConfig,
    )
    if err != nil {
        panic(err)
    }
    
    // 4. 閰嶇疆
    config := &sendsms.Config{
        PrimaryProvider: "aliyun",
        BackupProviders: []string{"tencent"},
        DefaultSign:     "鎮ㄧ殑绛惧悕",
        
        CacheConfig: &sendsms.CacheConfig{
            Prefix:           "sms:",
            VerificationExp:   5 * time.Minute,
            EnableLimit:      true,
            LimitCount:       5,
            LimitWindow:      time.Hour,
        },
        
        // 閲嶈瘯閰嶇疆
        RetryStrategy:    sendsms.RetryExponentialBackoff,
        RetryTimes:       3,
        RetryDelay:       1 * time.Second,
        MaxRetryDelay:    10 * time.Second,
        RetryMultiplier:  2.0,
        
        // Failover 閰嶇疆
        EnableFailover:      true,
        FailoverStrategy:    sendsms.FailoverSequential,
        FailoverCooldown:    5 * time.Minute,
        HealthCheckInterval: 1 * time.Minute,
        
        // 璇锋眰閰嶇疆
        Timeout:         10 * time.Second,
        BatchSize:       100,
        ConcurrentLimit: 10,
    }
    
    // 5. 鍒涘缓 SMS 瀹㈡埛绔?
    client, err := sendsms.NewSMSClient(
        "aliyun",
        []string{"tencent"},
        providers,
        rdb,
        config,
    )
    if err != nil {
        panic(err)
    }
    
    // 6. 鍙戦€佸崟鏉＄煭淇★紙鑷姩瀹归敊鍜?Failover锛?
    resp, err := client.Send(&sendsms.SMSRequest{
        Phone:    "13800138000",
        Template: "SMS_123456789",
        Params:   []string{"1234", "5鍒嗛挓"},
        SignName: "your_sign_name",
        Type:     sendsms.SMSNotification,
    })
    
    // 7. 鍙戦€侀獙璇佺爜
    resp, err = client.SendVerificationCode(&sendsms.VerificationCodeRequest{
        Phone:      "13800138000",
        ExpireTime: 5 * time.Minute,
        CodeLength: 6,
        Template:   "SMS_123456789",
        SignName:   "your_sign_name",
    })
    
    // 8. 楠岃瘉楠岃瘉鐮?
    result, err := client.VerifyCode(&sendsms.VerifyCodeRequest{
        Phone:     "13800138000",
        Code:      "123456",
        CleanOnce: true,
    })
    
    // 9. 鎵归噺鍙戦€侊紙甯﹀閿欙級
    batchReqs := []*sendsms.SMSRequest{
        {
            Phone:    "13800138000",
            Template: "SMS_123456789",
            Params:   []string{"1234"},
            SignName: "your_sign_name",
        },
        {
            Phone:    "13800138001",
            Template: "SMS_123456789",
            Params:   []string{"5678"},
            SignName: "your_sign_name",
        },
    }
    
    batchResult, err := client.SendBatch(batchReqs)
    if err != nil {
        // 澶勭悊閿欒
    }
    
    fmt.Printf("Total: %d, Success: %d, Failed: %d\n",
        batchResult.Total, batchResult.Success, batchResult.Failed)
    
    // 10. 閲嶈瘯澶辫触鐨勮姹?
    if len(batchResult.FailedReqs) > 0 {
        retryResult, err := client.RetryFailed(batchResult.FailedReqs)
        // 澶勭悊閲嶈瘯缁撴灉
    }
    
    // 11. 鑾峰彇鍋ュ悍鐘舵€?
    healthStatus := client.GetHealthStatus()
    for _, health := range healthStatus {
        fmt.Printf("Provider: %s, Healthy: %v, FailoverCount: %d\n",
            health.Name, health.IsHealthy, health.FailoverCount)
    }
}
```

### 楂樼骇浣跨敤绀轰緥

```go
// 浣跨敤妯℃澘鍙戦€侊紙绠€鍖栨帴鍙ｏ級
resp, err := client.SendWithTemplate(
    "13800138000",
    "SMS_123456789",
    "your_sign_name",
    []string{"鍙傛暟1", "鍙傛暟2"},
)

// 骞跺彂鎵归噺鍙戦€?
batchResult, err := client.SendBatchConcurrent(reqs, 10) // 10涓苟鍙?

// 鑾峰彇鏈嶅姟鍟嗕綑棰?
balance, err := providers["aliyun"].GetBalance()
fmt.Printf("Balance: %.2f\n", balance.Amount)

// 鍋ュ悍妫€鏌?
isHealthy := providers["aliyun"].HealthCheck()

// 鎵嬪姩鏍囪鏈嶅姟鍟嗗仴搴?涓嶅仴搴?
failoverMgr := client.GetFailoverManager()
failoverMgr.MarkProviderFailed("aliyun")
failoverMgr.MarkProviderHealthy("tencent")
```

## 瀹炵幇浼樺厛绾?

### Phase 1: 鏍稿績妗嗘灦锛堥珮浼樺厛绾э級
- [x] 鍒涘缓鐩綍缁撴瀯
- [ ] 瀹炵幇 types.go
- [ ] 瀹炵幇 interface.go
- [ ] 瀹炵幇 cache.go
- [ ] 瀹炵幇 config.go
- [ ] 瀹炵幇 errors.go
- [ ] 瀹炵幇 retry.go
- [ ] 瀹炵幇 failover.go
- [ ] 瀹炵幇 factory.go
- [ ] 瀹炵幇 sms.go

### Phase 2: 鏈嶅姟鍟嗗疄鐜帮紙涓紭鍏堢骇锛?
- [ ] 闃块噷浜?(Aliyun)
- [ ] 鑵捐浜?(Tencent)
- [ ] 鐧惧害浜?(Baidu)
- [ ] 鍗庝负浜?(Huawei)
- [ ] 缃戞槗浜戜俊 (Netease)
- [ ] 瀹硅仈浜?(Ronglian)

### Phase 3: 鎵╁睍鏈嶅姟鍟嗭紙浣庝紭鍏堢骇锛?
- [ ] 鏋佸厜 (Aurora)
- [ ] 鍒涜摑253 (Chuanglan)
- [ ] Twilio (鍥介檯)

### Phase 4: 娴嬭瘯鍜屾枃妗?
- [ ] 缂栧啓鍗曞厓娴嬭瘯
- [ ] 缂栧啓闆嗘垚娴嬭瘯
- [ ] 缂栧啓瀹归敊娴嬭瘯
- [ ] 缂栧啓 Failover 娴嬭瘯
- [ ] 瀹屽杽 README.md
- [ ] 娣诲姞浣跨敤绀轰緥
- [ ] 鎬ц兘娴嬭瘯

## 鎶€鏈鐐?

### 1. Redis 鎺ュ彛璁捐
- 浣跨敤 `redis.Cmdable` 鎺ュ彛浣滀负鍙傛暟绫诲瀷
- 鏀寔 `redis.Client` 鍜?`redis.ClusterClient`
- 鏀寔浜嬪姟鍜岀閬撴搷浣?
- 浣跨敤 Lua 渚ф湰淇濊瘉鍘熷瓙鎬?

### 2. 骞跺彂瀹夊叏
- 浣跨敤 Redis 鍘熷瓙鎿嶄綔
- 鍔犻攣鏈哄埗锛堝闇€瑕侊級
- 杩炴帴姹犵鐞?
- 浣跨敤 sync.RWMutex 淇濇姢鍏变韩鐘舵€?

### 3. 闄愭祦瀹炵幇
- 鍩轰簬 Redis Sliding Window
- 浣跨敤 Lua 渚ф湰淇濊瘉鍘熷瓙鎬?
- 鏀寔鍒嗗竷寮忕幆澧?
- 鍙厤缃檺娴佺瓥鐣?

### 4. 楠岃瘉鐮佺敓鎴?
- 浣跨敤 crypto/rand 淇濊瘉闅忔満鎬?
- 鏀寔鏁板瓧鍜屽瓧姣嶇粍鍚?
- 鍙厤缃暱搴﹀拰瀛楃闆?
- 閬垮厤閲嶅鐢熸垚

### 5. 閿欒澶勭悊
- 鍖呰鏈嶅姟鍟嗗師濮嬮敊璇?
- 鎻愪緵鍙嬪ソ鐨勯敊璇俊鎭?
- 鏀寔閿欒鍒嗙被
- 閿欒鍙噸璇曟€у垽鏂?

### 6. 鏃ュ織璁板綍
- 璁板綍鍙戦€佹垚鍔?澶辫触
- 璁板綍楠岃瘉鐮佹搷浣?
- 璁板綍闄愭祦瑙﹀彂
- 璁板綍 Failover 鍒囨崲
- 璁板綍閲嶈瘯灏濊瘯

### 7. 鎬ц兘浼樺寲
- 杩炴帴姹犲鐢?
- 鎵归噺鎿嶄綔浼樺寲
- 骞跺彂鎺у埗
- 缂撳瓨绛栫暐
- 鍑忓皯缃戠粶寰€杩?

### 8. 鐩戞帶鍜屽憡璀?
- 鏈嶅姟鍟嗗仴搴风姸鎬佺洃鎺?
- 鍙戦€佹垚鍔熺巼缁熻
- Failover 娆℃暟缁熻
- 閿欒鐜囩洃鎺?
- 鎬ц兘鎸囨爣鏀堕泦

## 渚濊禆绠＄悊

```go
require (
    github.com/aliyun/alibaba-cloud-sdk-go v3.0.0+incompatible
    github.com/tencentcloud/tencentcloud-sdk-go v3.x.x
    github.com/baiducloud/bce-sdk-go v0.x.x
    github.com/redis/go-redis/v9 v9.x.x
    github.com/google/uuid v1.x.x
)
```

## 娉ㄦ剰浜嬮」

### 1. 瀹夊叏鎬?
- AccessKey 绛夋晱鎰熶俊鎭笉瑕佺‖缂栫爜
- 浣跨敤鐜鍙橀噺鎴栭厤缃枃浠?
- 鑰冭檻鍔犲瘑瀛樺偍
- 浣跨敤 TLS 鍔犲瘑浼犺緭
- 闄愬埗 Redis 璁块棶鏉冮檺

### 2. 鎬ц兘
- 鍚堢悊閰嶇疆杩炴帴姹犲ぇ灏?
- 鎵归噺鎿嶄綔浼樺寲
- 缂撳瓨绛栫暐鍚堢悊
- 閬垮厤棰戠箒鍒涘缓杩炴帴
- 浣跨敤杩炴帴澶嶇敤

### 3. 鍙潬鎬?
- 閲嶈瘯鏈哄埗閰嶇疆鍚堢悊
- 闄嶇骇鏂规鍑嗗
- 鐩戞帶鍛婅瀹屽杽
- 鏃ュ織璁板綍璇︾粏
- 瀹氭湡鍋ュ悍妫€鏌?

### 4. 鍚堣鎬?
- 鐭俊鍐呭瀹℃牳
- 鐢ㄦ埛闅愮淇濇姢
- 閬靛畧褰撳湴娉曡
- 澶囨淇℃伅鍑嗙‘
- 棰戠巼鎺у埗鍚堢悊

### 5. 鎴愭湰鎺у埗
- 鍚堢悊閫夋嫨鏈嶅姟鍟?
- 鐩戞帶鍙戦€侀噺
- 鎵归噺鍙戦€侀檷浣庢垚鏈?
- 浼樺寲妯℃澘浣跨敤
- 閬垮厤閲嶅鍙戦€?

### 6. 瀹归敊閰嶇疆寤鸿
- 閲嶈瘯娆℃暟: 2-3 娆?
- 閲嶈瘯寤惰繜: 1-5 绉?
- Failover 鍐峰嵈鏃堕棿: 5-10 鍒嗛挓
- 鍋ュ悍妫€鏌ラ棿闅? 1-5 鍒嗛挓
