package environment

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/andynikk/advancedmetrics/internal/networks"
	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type AgentConfigENV struct {
	Address        string        `env:"ADDRESS" envDefault:"localhost:8080"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL" envDefault:"10s"`
	PollInterval   time.Duration `env:"POLL_INTERVAL" envDefault:"2s"`
	Key            string        `env:"KEY"`
	CryptoKey      string        `env:"CRYPTO_KEY"`
	Config         string        `env:"CONFIG"`
}

type AgentConfig struct {
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
	Key            string
	CryptoKey      string
	ConfigFilePath string
	IPAddress      string
}

type AgentConfigFile struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

type ServerConfigENV struct {
	Address       string        `env:"ADDRESS" envDefault:"localhost:8080"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" envDefault:"300s"`
	StoreFile     string        `env:"STORE_FILE" envDefault:"/tmp/devops-metrics-db.json"`
	Restore       bool          `env:"RESTORE" envDefault:"true"`
	Key           string        `env:"KEY"`
	DatabaseDsn   string        `env:"DATABASE_DSN"`
	CryptoKey     string        `env:"CRYPTO_KEY"`
	Config        string        `env:"CONFIG"`
	TrustedSubnet string        `env:"TRUSTED_SUBNET"`
}

type ServerConfig struct {
	StoreInterval      time.Duration
	StoreFile          string
	Restore            bool
	Address            string
	Key                string
	DatabaseDsn        string
	TypeMetricsStorage repository.MapTypeStore
	CryptoKey          string
	ConfigFilePath     string
	TrustedSubnet      *net.IPNet
}

type ServerConfigFile struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDsn   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
	TrustedSubnet string `json:"trusted_subnet"`
}

func ThisOSWindows() bool {

	var stderr bytes.Buffer
	defer stderr.Reset()

	var out bytes.Buffer
	defer out.Reset()

	cmd := exec.Command("cmd", "ver")
	cmd.Stdin = strings.NewReader("some input")
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return false
	}
	myOS := out.String()
	if strings.Contains(myOS, "Microsoft Windows") {
		return true
	}
	return false
}

func ParsCfgByte(res []byte) bytes.Buffer {

	var out bytes.Buffer
	configLines := strings.Split(string(res), "\n")
	for i := 0; i < len(configLines); i++ {

		if configLines[i] != "" {
			var strs string
			splitStr := strings.SplitAfterN(configLines[i], "// ", -1)
			if len(splitStr) != 0 {
				strs = strings.Replace(splitStr[0], "// ", "\n", -1)
				out.WriteString(strs)
			}
		}
	}
	return out
}

func GetAgentConfigFile(file *string) AgentConfigFile {
	var sConfig AgentConfigFile

	res, err := os.ReadFile(*file)
	if err != nil {
		return sConfig
	}

	out := ParsCfgByte(res)
	defer out.Reset()

	if err = json.Unmarshal([]byte(out.String()), &sConfig); err != nil {
		return sConfig
	}
	if ThisOSWindows() {
		sConfig.CryptoKey = strings.Replace(sConfig.CryptoKey, "/", "\\", -1)
	}

	return sConfig

}

func InitConfigAgent() *AgentConfig {
	configAgent := AgentConfig{}
	configAgent.InitConfigAgentENV()
	configAgent.InitConfigAgentFlag()
	configAgent.InitConfigAgentFile()
	configAgent.InitConfigAgentDefault()

	return &configAgent
}

func (ac *AgentConfig) InitConfigAgentENV() {

	var cfgENV AgentConfigENV
	err := env.Parse(&cfgENV)
	if err != nil {
		log.Fatal(err)
	}

	pathFileCfg := ""
	if _, ok := os.LookupEnv("CONFIG"); ok {
		pathFileCfg = cfgENV.Config
	}

	addressServ := ""
	if _, ok := os.LookupEnv("ADDRESS"); ok {
		addressServ = cfgENV.Address
	}

	var reportIntervalMetric time.Duration
	if _, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		reportIntervalMetric = cfgENV.ReportInterval
	}

	var pollIntervalMetrics time.Duration
	if _, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		pollIntervalMetrics = cfgENV.PollInterval
	}

	keyHash := ""
	if _, ok := os.LookupEnv("KEY"); ok {
		keyHash = cfgENV.Key
	}

	patchCryptoKey := ""
	if _, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		patchCryptoKey = cfgENV.CryptoKey
	}

	ac.Address = addressServ
	ac.ReportInterval = reportIntervalMetric
	ac.PollInterval = pollIntervalMetrics
	ac.Key = keyHash
	ac.CryptoKey = patchCryptoKey
	ac.ConfigFilePath = pathFileCfg
}

func (ac *AgentConfig) InitConfigAgentFlag() {

	addressPtr := flag.String("a", "", "имя сервера")
	reportIntervalPtr := flag.Duration("r", 0, "интервал отправки на сервер")
	pollIntervalPtr := flag.Duration("p", 0, "интервал сбора метрик")
	keyFlag := flag.String("k", "", "ключ хеширования")
	cryptoKeyFlag := flag.String("crypto-key", "", "файл с криптоключем")
	fileCfg := flag.String("config", "", "файл с конфигурацией")
	fileCfgC := flag.String("c", "", "файл с конфигурацией")

	flag.Parse()

	pathFileCfg := ""
	if *fileCfg != "" {
		pathFileCfg = *fileCfg
	} else if *fileCfgC != "" {
		pathFileCfg = *fileCfgC
	}

	if ac.Address == "" {
		ac.Address = *addressPtr
	}
	if ac.ReportInterval == 0 {
		ac.ReportInterval = *reportIntervalPtr
	}
	if ac.PollInterval == 0 {
		ac.PollInterval = *pollIntervalPtr
	}
	if ac.Key == "" {
		ac.Key = *keyFlag
	}
	if ac.CryptoKey == "" {
		ac.CryptoKey = *cryptoKeyFlag
	}
	if ac.ConfigFilePath == "" {
		ac.ConfigFilePath = pathFileCfg
	}
}

func (ac *AgentConfig) InitConfigAgentFile() {

	if ac.ConfigFilePath == "" {
		return
	}

	var jsonCfg AgentConfigFile
	jsonCfg = GetAgentConfigFile(&ac.ConfigFilePath)

	addressServ := jsonCfg.Address
	reportIntervalMetric, _ := time.ParseDuration(jsonCfg.ReportInterval)
	pollIntervalMetrics, _ := time.ParseDuration(jsonCfg.PollInterval)
	patchCryptoKey := jsonCfg.CryptoKey

	if ac.Address == "" {
		ac.Address = addressServ
	}
	if ac.ReportInterval == 0 {
		ac.ReportInterval = reportIntervalMetric
	}
	if ac.PollInterval == 0 {
		ac.PollInterval = pollIntervalMetrics
	}
	if ac.CryptoKey == "" {
		ac.CryptoKey = patchCryptoKey
	}
	if ac.CryptoKey == "" {
		ac.CryptoKey = patchCryptoKey
	}
}

func (ac *AgentConfig) InitConfigAgentDefault() {

	addressServ := constants.AddressServer
	reportIntervalMetric := constants.ReportInterval * time.Second
	pollIntervalMetrics := constants.PollInterval * time.Second

	if ac.Address == "" {
		ac.Address = addressServ
	}
	if ac.ReportInterval == 0 {
		ac.ReportInterval = reportIntervalMetric
	}
	if ac.PollInterval == 0 {
		ac.PollInterval = pollIntervalMetrics
	}

	hn, _ := os.Hostname()
	IPs, _ := net.LookupIP(hn)
	ac.IPAddress = networks.IPStr(IPs)
}

func GetServerConfigFile(file *string) ServerConfigFile {
	var sConfig ServerConfigFile

	res, err := os.ReadFile(*file)
	if err != nil {
		return sConfig
	}

	out := ParsCfgByte(res)
	defer out.Reset()

	if err = json.Unmarshal([]byte(out.String()), &sConfig); err != nil {
		return sConfig
	}
	if ThisOSWindows() {
		sConfig.CryptoKey = strings.Replace(sConfig.CryptoKey, "/", "\\", -1)
		sConfig.StoreFile = strings.Replace(sConfig.StoreFile, "/", "\\", -1)
	}

	return sConfig

}

func InitConfigServer() *ServerConfig {
	constants.Logger.Log = zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	sc := ServerConfig{}
	sc.InitConfigServerENV()
	sc.InitConfigServerFlag()
	sc.InitConfigServerFile()
	sc.InitConfigServerDefault()

	return &sc
}

func (sc *ServerConfig) InitConfigServerENV() {

	var cfgENV ServerConfigENV
	err := env.Parse(&cfgENV)
	if err != nil {
		return
	}

	var addressServ string
	if _, ok := os.LookupEnv("ADDRESS"); ok {
		addressServ = cfgENV.Address
	}

	var restoreMetric bool
	if _, ok := os.LookupEnv("RESTORE"); ok {
		restoreMetric = cfgENV.Restore
	}

	var storeIntervalMetrics time.Duration
	if _, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		storeIntervalMetrics = cfgENV.StoreInterval
	}

	var storeFileMetrics string
	if _, ok := os.LookupEnv("STORE_FILE"); ok {
		storeFileMetrics = cfgENV.StoreFile
	}

	keyHash := cfgENV.Key
	if _, ok := os.LookupEnv("KEY"); !ok {
		keyHash = cfgENV.Key
	}

	var databaseDsn string
	if _, ok := os.LookupEnv("DATABASE_DSN"); ok {
		databaseDsn = cfgENV.DatabaseDsn
	}

	var patchCryptoKey string
	if _, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		patchCryptoKey = cfgENV.CryptoKey
	}

	var patchFileConfig string
	if _, ok := os.LookupEnv("CONFIG"); ok {
		patchFileConfig = cfgENV.Config
	}

	var trustedSubnet string
	if _, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		trustedSubnet = cfgENV.TrustedSubnet
	}

	MapTypeStore := make(repository.MapTypeStore)
	if databaseDsn != "" {
		typeDB := repository.TypeStoreDataDB{}
		MapTypeStore[constants.MetricsStorageDB.String()] = &typeDB
	} else if storeFileMetrics != "" {
		typeFile := repository.TypeStoreDataFile{}
		MapTypeStore[constants.MetricsStorageFile.String()] = &typeFile
	}

	sc.StoreInterval = storeIntervalMetrics
	sc.StoreFile = storeFileMetrics
	sc.Restore = restoreMetric
	sc.Address = addressServ
	sc.Key = keyHash
	sc.DatabaseDsn = databaseDsn
	sc.TypeMetricsStorage = MapTypeStore
	sc.CryptoKey = patchCryptoKey
	sc.ConfigFilePath = patchFileConfig

	_, ipv4Net, _ := net.ParseCIDR(trustedSubnet)
	sc.TrustedSubnet = ipv4Net
}

func (sc *ServerConfig) InitConfigServerFlag() {

	addressPtr := flag.String("a", "", "имя сервера")
	restorePtr := flag.Bool("r", false, "восстанавливать значения при старте")
	storeIntervalPtr := flag.Duration("i", 0, "интервал автосохранения (сек.)")
	storeFilePtr := flag.String("f", "", "путь к файлу метрик")
	keyFlag := flag.String("k", "", "ключ хеша")
	keyDatabaseDsn := flag.String("d", "", "строка соединения с базой")
	cryptoKeyFlag := flag.String("crypto-key", "", "файл с криптоключем")
	fileCfg := flag.String("config", "", "файл с конфигурацией")
	fileCfgC := flag.String("c", "", "файл с конфигурацией")
	trustedSubnet := flag.String("t", "", "строковое представление бесклассовой адресации (CIDR)")

	flag.Parse()

	pathFileCfg := ""
	if *fileCfg != "" {
		pathFileCfg = *fileCfg
	} else if *fileCfgC != "" {
		pathFileCfg = *fileCfgC
	}

	MapTypeStore := make(repository.MapTypeStore)
	if len(sc.TypeMetricsStorage) == 0 {
		if *keyDatabaseDsn != "" {
			typeDB := repository.TypeStoreDataDB{}
			MapTypeStore[constants.MetricsStorageDB.String()] = &typeDB
		} else if *cryptoKeyFlag != "" {
			typeFile := repository.TypeStoreDataFile{}
			MapTypeStore[constants.MetricsStorageFile.String()] = &typeFile
		}
	}

	if sc.Address == "" {
		sc.Address = *addressPtr
	}
	if sc.StoreInterval == 0 {
		sc.StoreInterval = *storeIntervalPtr
	}
	if sc.StoreFile == "" {
		sc.StoreFile = *storeFilePtr
	}
	if !sc.Restore {
		sc.Restore = *restorePtr
	}
	if sc.Key == "" {
		sc.Key = *keyFlag
	}
	if sc.DatabaseDsn == "" {
		sc.DatabaseDsn = *keyDatabaseDsn
	}
	if sc.CryptoKey == "" {
		sc.CryptoKey = *cryptoKeyFlag
	}
	if sc.ConfigFilePath == "" {
		sc.ConfigFilePath = pathFileCfg
	}
	if len(sc.TypeMetricsStorage) == 0 {
		sc.TypeMetricsStorage = MapTypeStore
	}
	if sc.TrustedSubnet.String() == "" {
		_, ipv4Net, _ := net.ParseCIDR(*trustedSubnet)
		sc.TrustedSubnet = ipv4Net
	}
}

func (sc *ServerConfig) InitConfigServerFile() {

	if sc.ConfigFilePath == "" {
		return
	}

	var jsonCfg ServerConfigFile
	jsonCfg = GetServerConfigFile(&sc.ConfigFilePath)

	addressServ := jsonCfg.Address
	restoreMetric := jsonCfg.Restore
	storeIntervalMetrics, _ := time.ParseDuration(jsonCfg.StoreInterval)
	storeFileMetrics := jsonCfg.StoreFile
	databaseDsn := jsonCfg.DatabaseDsn
	patchCryptoKey := jsonCfg.CryptoKey
	trustedSubnet := jsonCfg.TrustedSubnet

	MapTypeStore := make(repository.MapTypeStore)
	if len(sc.TypeMetricsStorage) == 0 {
		if databaseDsn != "" {
			typeDB := repository.TypeStoreDataDB{}
			MapTypeStore[constants.MetricsStorageDB.String()] = &typeDB
		} else if storeFileMetrics != "" {
			typeFile := repository.TypeStoreDataFile{}
			MapTypeStore[constants.MetricsStorageFile.String()] = &typeFile
		}
	}

	if sc.Address == "" {
		sc.Address = addressServ
	}
	if sc.StoreInterval == 0 {
		sc.StoreInterval = storeIntervalMetrics
	}
	if sc.StoreFile == "" {
		sc.StoreFile = storeFileMetrics
	}
	if !sc.Restore {
		sc.Restore = restoreMetric
	}
	if sc.DatabaseDsn == "" {
		sc.DatabaseDsn = databaseDsn
	}
	if sc.CryptoKey == "" {
		sc.CryptoKey = patchCryptoKey
	}
	if len(sc.TypeMetricsStorage) == 0 {
		sc.TypeMetricsStorage = MapTypeStore
	}
	if sc.TrustedSubnet.String() == "" {
		_, ipv4Net, _ := net.ParseCIDR(trustedSubnet)
		sc.TrustedSubnet = ipv4Net
	}
}

func (sc *ServerConfig) InitConfigServerDefault() {

	if sc.Address == "" {
		sc.Address = constants.AddressServer
	}
	if sc.StoreInterval == 0 {
		sc.StoreInterval = constants.StoreInterval
	}
	if sc.StoreFile == "" {
		sc.StoreFile = constants.StoreFile
	}
	if !sc.Restore {
		sc.Restore = constants.Restore
	}
}
