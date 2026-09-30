package config

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Jinnrry/pmail/utils/context"
	"github.com/Jinnrry/pmail/utils/errors"
	"github.com/Jinnrry/pmail/utils/file"
	log "github.com/sirupsen/logrus"
)

var IsInit bool

// DefaultDKIMSelector is the selector used for keys generated before per-domain
// DKIM existed. Keeping it as the default preserves already published DNS records.
const DefaultDKIMSelector = "default"

// Domain is a single mail domain served by this instance. Every domain owns its
// DKIM material so that mail sent from different domains stays DMARC aligned.
type Domain struct {
	Name               string `json:"name"`
	DKIMSelector       string `json:"dkimSelector,omitempty"`
	DKIMPrivateKeyPath string `json:"dkimPrivateKeyPath,omitempty"`
}

// NewDomain builds a domain with default DKIM settings applied by normalize.
func NewDomain(name string) Domain {
	return Domain{Name: strings.ToLower(strings.TrimSpace(name))}
}

// UnmarshalJSON accepts both the historical plain string form ("example.com")
// and the structured object form, so existing config files keep working.
func (d *Domain) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		d.Name = name
		return nil
	}
	type plain Domain
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*d = Domain(p)
	return nil
}

type Config struct {
	LogLevel             string            `json:"logLevel"` // 日志级别
	Domain               string            `json:"domain"`
	Domains              []Domain          `json:"domains"` //多域名设置，把所有收信域名都填进去，第一个为主域名
	WebDomain            string            `json:"webDomain"`
	DkimPrivateKeyPath   string            `json:"dkimPrivateKeyPath"` // 主域名DKIM私钥，兼容旧配置
	SSLType              string            `json:"sslType"`            // 0表示自动生成证书，HTTP挑战模式，1表示用户上传证书，2表示自动-DNS挑战模式
	SSLPrivateKeyPath    string            `json:"SSLPrivateKeyPath"`
	SSLPublicKeyPath     string            `json:"SSLPublicKeyPath"`
	DbDSN                string            `json:"dbDSN"`
	DbType               string            `json:"dbType"`
	HttpsEnabled         int               `json:"httpsEnabled"`    //后台页面是否启用https，0默认（启用），1启用，2不启用
	SpamFilterLevel      int               `json:"spamFilterLevel"` //垃圾邮件过滤级别，0不过滤、1 spf dkim 校验均失败时过滤，2 spf校验不通过时过滤 3,dkim 校验不过的时候过滤
	HttpPort             int               `json:"httpPort"`        //http服务端口设置，默认80
	HttpsPort            int               `json:"httpsPort"`       //https服务端口，默认443
	WeChatPushAppId      string            `json:"weChatPushAppId"`
	WeChatPushSecret     string            `json:"weChatPushSecret"`
	WeChatPushTemplateId string            `json:"weChatPushTemplateId"`
	WeChatPushUserId     string            `json:"weChatPushUserId"`
	TgBotToken           string            `json:"tgBotToken"`
	TgChatId             string            `json:"tgChatId"`
	IsInit               bool              `json:"isInit"`
	WebPushUrl           string            `json:"webPushUrl"`
	WebPushToken         string            `json:"webPushToken"`
	Tables               map[string]string `json:"-"`
	TablesInitData       map[string]string `json:"-"`
}

var ROOT_PATH = ""

func init() {
	envs := os.Environ()
	for _, env := range envs {
		if strings.HasPrefix(env, "PMail_ROOT=") {
			ROOT_PATH = strings.TrimSpace(strings.ReplaceAll(env, "PMail_ROOT=", ""))
			if !strings.HasSuffix(ROOT_PATH, "/") {
				ROOT_PATH += "/"
			}

			fmt.Println("Env Root Path:", ROOT_PATH)
			return
		}
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	realPath, err := filepath.EvalSymlinks(exPath)
	if err != nil {
		panic(err)
	}
	// 如果是Goland运行，不修改根路径
	if strings.Contains(realPath, "GoLand") && strings.Contains(realPath, "JetBrains") {
		return
	}

	if !strings.HasSuffix(realPath, "/") {
		realPath += "/"
	}
	ROOT_PATH = realPath
	fmt.Println("Root Path:", ROOT_PATH)
}

const DBTypeMySQL = "mysql"
const DBTypeSQLite = "sqlite"
const DBTypePostgres = "postgres"
const SSLTypeAutoHTTP = "0" //自动生成证书
const SSLTypeAutoDNS = "2"  //自动生成证书，DNS api验证
const SSLTypeUser = "1"     //用户上传证书

var DBTypes []string = []string{DBTypeMySQL, DBTypeSQLite, DBTypePostgres}

// The setup wizard runs on a throwaway port before the real listeners start, so
// the port is process state rather than part of the persisted configuration.
var setupPort int

func GetSetupPort() int     { return setupPort }
func SetSetupPort(port int) { setupPort = port }

var emptyConfig = &Config{}

// current holds the published configuration snapshot. Readers must go through
// Get; writers must publish a whole new snapshot with Set so that concurrent
// readers never observe a half-updated configuration.
var current atomic.Pointer[Config]

// Get returns the current configuration snapshot. The result is read-only:
// clone it, change the clone, then publish it with Set.
func Get() *Config {
	if cfg := current.Load(); cfg != nil {
		return cfg
	}
	return emptyConfig
}

// Set normalizes cfg and publishes it as the new snapshot.
func Set(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.normalize()
	current.Store(cfg)
}

type logFormatter struct {
}

// Format 定义日志输出格式
func (l *logFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := bytes.Buffer{}

	b.WriteString(fmt.Sprintf("[%s]", entry.Level.String()))
	b.WriteString(fmt.Sprintf("[%s]", entry.Time.Format("2006-01-02 15:04:05")))
	if entry.Context != nil {
		if ctx, ok := entry.Context.(*context.Context); ok && ctx != nil {
			b.WriteString(fmt.Sprintf("[%s]", ctx.GetValue(context.LogID)))
		}
	}
	if entry.Caller != nil {
		b.WriteString(fmt.Sprintf("[%s:%d]", entry.Caller.File, entry.Caller.Line))
	}
	b.WriteString(entry.Message)

	b.WriteString("\n")
	return b.Bytes(), nil
}

// ConfigPath is the location of the persisted configuration file.
func ConfigPath() string {
	return filepath.Join(ROOT_PATH, "config", "config.json")
}

// Init loads config.json and publishes it. On failure the previously published
// snapshot is kept so a typo in the file cannot take the server down.
func Init() {
	cfg, err := ReadConfig()
	if err != nil {
		log.Errorf("config file not found or invalid, keeping previous config: %s", err.Error())
		return
	}
	Set(cfg)
	if cfg.Domain != "" && cfg.IsInit {
		IsInit = true
	}
	configureLogging(cfg)
}

// Reload re-reads config.json from disk and publishes it. Running listeners keep
// serving: only state that is read through Get (domains, DKIM, log level) changes.
func Reload() error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	Set(cfg)
	if cfg.Domain != "" && cfg.IsInit {
		IsInit = true
	}
	configureLogging(cfg)
	return nil
}

func configureLogging(cfg *Config) {
	// 设置日志格式为json格式
	// ReportCaller must be enabled before installing the formatter, which reads entry.Caller.
	log.SetReportCaller(true)
	log.SetFormatter(&logFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr,标准错误）
	// 日志消息输出可以是任意的io.writer类型
	log.SetOutput(os.Stdout)

	var cstZone = time.FixedZone("CST", 8*3600)
	time.Local = cstZone

	switch cfg.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

func ReadPrivateKey() (*ecdsa.PrivateKey, bool) {
	key, err := os.ReadFile(ROOT_PATH + "./config/ssl/account_private.pem")
	if err != nil {
		return createNewPrivateKey(), true
	}

	block, _ := pem.Decode(key)
	x509Encoded := block.Bytes
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)

	return privateKey, false
}

func createNewPrivateKey() *ecdsa.PrivateKey {
	// Create a user. New accounts need an email and private key to start.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)

	// 将ec 密钥写入到 pem文件里
	keypem, _ := os.OpenFile(ROOT_PATH+"./config/ssl/account_private.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	err = pem.Encode(keypem, &pem.Block{Type: "EC PRIVATE KEY", Bytes: x509Encoded})
	if err != nil {
		panic(err)
	}
	return privateKey
}

// WriteConfig persists cfg atomically so that a crash or a concurrent reader
// never sees a truncated file. Paths are stored relative to ROOT_PATH to keep the
// file portable.
func WriteConfig(cfg *Config) error {
	if cfg == nil {
		return errors.New("config must not be nil")
	}

	clone := cfg.Clone()
	// Normalize first: it fills in defaults such as a domain's DKIM key path.
	clone.normalize()
	// Then store portable paths, so the file stays independent of the install directory.
	clone.pathsForStorage()

	data, err := json.MarshalIndent(clone, "", "  ")
	if err != nil {
		return errors.Wrap(err)
	}

	// config.json carries the database password and push tokens.
	if err := file.WriteAtomic(ConfigPath(), data, 0600); err != nil {
		return errors.Wrap(err)
	}
	return nil
}

func ReadConfig() (*Config, error) {
	configData := Config{
		DkimPrivateKeyPath: filepath.Join(ROOT_PATH, "config", "dkim", "dkim.priv"),
		SSLPrivateKeyPath:  filepath.Join(ROOT_PATH, "config", "ssl", "private.key"),
		SSLPublicKeyPath:   filepath.Join(ROOT_PATH, "config", "ssl", "public.crt"),
	}

	path := ConfigPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("Read Config Error:%s", err.Error())
			return nil, errors.Wrap(err)
		}
		if err := WriteConfig(&configData); err != nil {
			log.Errorf("Write Config Error:%s", err.Error())
			return nil, err
		}
		configData.normalize()
		return &configData, nil
	}

	if err := json.Unmarshal(raw, &configData); err != nil {
		log.Errorf("Read Config Unmarshal Error:%s", err.Error())
		return nil, errors.Wrap(err)
	}
	configData.normalize()
	return &configData, nil
}

// Clone returns a deep enough copy to be mutated and published independently of
// the current snapshot.
func (c *Config) Clone() *Config {
	clone := *c
	clone.Domains = make([]Domain, len(c.Domains))
	copy(clone.Domains, c.Domains)
	return &clone
}

// PrimaryDomain is the domain used for the default sender address and for
// protocol greetings.
func (c *Config) PrimaryDomain() string {
	if c.Domain != "" {
		return c.Domain
	}
	if len(c.Domains) > 0 {
		return c.Domains[0].Name
	}
	return ""
}

// DomainNames lists every configured domain, primary first.
func (c *Config) DomainNames() []string {
	ret := make([]string, 0, len(c.Domains))
	for _, d := range c.Domains {
		ret = append(ret, d.Name)
	}
	return ret
}

// HasDomain reports whether name is a mail domain served by this instance.
// Matching is case insensitive.
func (c *Config) HasDomain(name string) bool {
	_, ok := c.FindDomain(name)
	return ok
}

// FindDomain returns the configuration of a served domain.
func (c *Config) FindDomain(name string) (Domain, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, d := range c.Domains {
		if d.Name == name {
			return d, true
		}
	}
	return Domain{}, false
}

// normalize enforces the invariants every reader relies on: lowercase, unique
// domain names with the primary domain first, and a DKIM key path for each.
func (c *Config) normalize() {
	c.fixPath()

	ordered := make([]Domain, 0, len(c.Domains)+1)
	seen := map[string]int{}
	add := func(d Domain) {
		d.Name = strings.ToLower(strings.TrimSpace(d.Name))
		if d.Name == "" {
			return
		}
		if i, ok := seen[d.Name]; ok {
			// Keep the richest definition when a domain is listed twice.
			if ordered[i].DKIMPrivateKeyPath == "" {
				ordered[i] = d
			}
			return
		}
		seen[d.Name] = len(ordered)
		ordered = append(ordered, d)
	}

	if c.Domain != "" {
		primary := Domain{Name: c.Domain}
		if existing, ok := lookupRaw(c.Domains, c.Domain); ok {
			primary = existing
		}
		add(primary)
	}
	for _, d := range c.Domains {
		add(d)
	}

	for i := range ordered {
		if ordered[i].DKIMSelector == "" {
			ordered[i].DKIMSelector = DefaultDKIMSelector
		}
		if ordered[i].DKIMPrivateKeyPath == "" {
			if i == 0 && c.DkimPrivateKeyPath != "" {
				ordered[i].DKIMPrivateKeyPath = c.DkimPrivateKeyPath
			} else {
				ordered[i].DKIMPrivateKeyPath = filepath.Join(ROOT_PATH, "config", "dkim", ordered[i].Name+".priv")
			}
		} else {
			ordered[i].DKIMPrivateKeyPath = absPath(ordered[i].DKIMPrivateKeyPath)
		}
	}

	c.Domains = ordered
	if c.Domain == "" && len(ordered) > 0 {
		c.Domain = ordered[0].Name
	}
	if c.DkimPrivateKeyPath == "" && len(ordered) > 0 {
		c.DkimPrivateKeyPath = ordered[0].DKIMPrivateKeyPath
	}
}

func lookupRaw(domains []Domain, name string) (Domain, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, d := range domains {
		if strings.ToLower(strings.TrimSpace(d.Name)) == name {
			return d, true
		}
	}
	return Domain{}, false
}

// fixPath turns the relative paths stored in config.json into runtime absolute
// paths. pathsForStorage is its inverse.
func (c *Config) fixPath() {
	if c.DbType == DBTypeSQLite {
		c.DbDSN = absPath(c.DbDSN)
	}
	c.SSLPublicKeyPath = absPath(c.SSLPublicKeyPath)
	c.SSLPrivateKeyPath = absPath(c.SSLPrivateKeyPath)
	c.DkimPrivateKeyPath = absPath(c.DkimPrivateKeyPath)
}

func (c *Config) pathsForStorage() {
	c.DbDSN = relPath(c.DbDSN)
	c.SSLPublicKeyPath = relPath(c.SSLPublicKeyPath)
	c.SSLPrivateKeyPath = relPath(c.SSLPrivateKeyPath)
	c.DkimPrivateKeyPath = relPath(c.DkimPrivateKeyPath)
	for i := range c.Domains {
		c.Domains[i].DKIMPrivateKeyPath = relPath(c.Domains[i].DKIMPrivateKeyPath)
	}
}

func absPath(p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(ROOT_PATH, p)
}

func relPath(p string) string {
	if p == "" || ROOT_PATH == "" || !filepath.IsAbs(p) {
		return p
	}
	rel, err := filepath.Rel(ROOT_PATH, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p
	}
	return rel
}
