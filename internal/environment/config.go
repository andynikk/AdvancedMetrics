package environment

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

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
}

type AgentConfigFile struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

type AgentConfigDefault struct {
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
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
}

type ServerConfigFile struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDsn   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
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

func InitConfigAgent() AgentConfig {

	addressPtr := flag.String("a", "", "имя сервера")
	reportIntervalPtr := flag.Duration("r", 0, "интервал отправки на сервер")
	pollIntervalPtr := flag.Duration("p", 0, "интервал сбора метрик")
	keyFlag := flag.String("k", "", "ключ хеширования")
	cryptoKeyFlag := flag.String("crypto-key", "", "файл с криптоключем")
	fileCfg := flag.String("config", "", "файл с конфигурацией")
	fileCfgC := flag.String("c", "", "файл с конфигурацией")

	flag.Parse()

	var cfgENV AgentConfigENV
	err := env.Parse(&cfgENV)
	if err != nil {
		log.Fatal(err)
	}

	pathFileCfg := ""
	if *fileCfg != "" {
		pathFileCfg = *fileCfg
	} else if *fileCfgC != "" {
		pathFileCfg = *fileCfgC
	}
	if _, ok := os.LookupEnv("CONFIG"); ok {
		pathFileCfg = cfgENV.Config
	} else {
		pathFileCfg = pathFileCfg
	}
	if pathFileCfg == "" {
		pathFileCfg = "c:\\Bases\\Go\\AdvancedMetrics\\cmd\\agent\\config.cfg"
	}

	var jsonCfg AgentConfigFile
	jsonCfg = GetAgentConfigFile(&pathFileCfg)

	addressServ := ""
	if _, ok := os.LookupEnv("ADDRESS"); ok {
		addressServ = cfgENV.Address
	} else if *addressPtr != "" {
		addressServ = *addressPtr
	} else if jsonCfg.Address != "" {
		addressServ = jsonCfg.Address
	} else {
		addressServ = constants.AddressServer
	}

	var reportIntervalMetric time.Duration
	if _, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		reportIntervalMetric = cfgENV.ReportInterval
	} else if *reportIntervalPtr != 0 {
		reportIntervalMetric = *reportIntervalPtr
	} else if rmi, _ := time.ParseDuration(jsonCfg.ReportInterval); rmi != 0 {
		reportIntervalMetric = rmi
	} else {
		reportIntervalMetric = constants.ReportInterval * time.Second
	}

	var pollIntervalMetrics time.Duration
	if _, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		pollIntervalMetrics = cfgENV.PollInterval
	} else if *pollIntervalPtr != 0 {
		pollIntervalMetrics = *pollIntervalPtr
	} else if pi, _ := time.ParseDuration(jsonCfg.PollInterval); pi != 0 {
		pollIntervalMetrics = pi
	} else {
		pollIntervalMetrics = constants.PollInterval * time.Second
	}

	keyHash := ""
	if _, ok := os.LookupEnv("KEY"); ok {
		keyHash = cfgENV.Key
	} else {
		keyHash = *keyFlag
	}

	patchCryptoKey := ""
	if _, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		patchCryptoKey = cfgENV.CryptoKey
	} else if *cryptoKeyFlag != "" {
		patchCryptoKey = *cryptoKeyFlag
	} else {
		patchCryptoKey = jsonCfg.CryptoKey
	}

	return AgentConfig{
		Address:        addressServ,
		ReportInterval: reportIntervalMetric,
		PollInterval:   pollIntervalMetrics,
		Key:            keyHash,
		CryptoKey:      patchCryptoKey,
	}
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

func InitConfigServer() (ServerConfig, error) {

	addressPtr := flag.String("a", "", "имя сервера")
	restorePtr := flag.Bool("r", false, "восстанавливать значения при старте")
	storeIntervalPtr := flag.Duration("i", 0, "интервал автосохранения (сек.)")
	storeFilePtr := flag.String("f", "", "путь к файлу метрик")
	keyFlag := flag.String("k", "", "ключ хеша")
	keyDatabaseDsn := flag.String("d", "", "строка соединения с базой")
	cryptoKeyFlag := flag.String("crypto-key", "", "файл с криптоключем")
	fileCfg := flag.String("config", "", "файл с конфигурацией")
	fileCfgC := flag.String("c", "", "файл с конфигурацией")

	flag.Parse()

	var cfgENV ServerConfigENV
	err := env.Parse(&cfgENV)
	if err != nil {
		return ServerConfig{}, err
	}

	pathFileCfg := ""
	if *fileCfg != "" {
		pathFileCfg = *fileCfg
	} else if *fileCfgC != "" {
		pathFileCfg = *fileCfgC
	}
	if _, ok := os.LookupEnv("CONFIG"); ok {
		pathFileCfg = cfgENV.Config
	} else {
		pathFileCfg = pathFileCfg
	}
	if pathFileCfg == "" {
		pathFileCfg = "c:\\Bases\\Go\\AdvancedMetrics\\cmd\\server\\config.cfg"
	}

	var jsonCfg ServerConfigFile
	jsonCfg = GetServerConfigFile(&pathFileCfg)

	var addressServ string
	if _, ok := os.LookupEnv("ADDRESS"); ok {
		addressServ = cfgENV.Address
	} else if *addressPtr != "" {
		addressServ = *addressPtr
	} else if jsonCfg.Address != "" {
		addressServ = jsonCfg.Address
	} else {
		addressServ = constants.AddressServer
	}

	var restoreMetric bool
	if _, ok := os.LookupEnv("RESTORE"); ok {
		restoreMetric = cfgENV.Restore
	} else if *restorePtr {
		restoreMetric = *restorePtr
	} else if jsonCfg.Restore {
		restoreMetric = jsonCfg.Restore
	} else {
		restoreMetric = constants.Restore
	}

	var storeIntervalMetrics time.Duration
	if _, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		storeIntervalMetrics = cfgENV.StoreInterval
	} else if *storeIntervalPtr != 0 {
		storeIntervalMetrics = *storeIntervalPtr
	} else if si, _ := time.ParseDuration(jsonCfg.StoreInterval); si != 0 {
		storeIntervalMetrics = si
	} else {
		storeIntervalMetrics = constants.StoreInterval
	}

	var storeFileMetrics string
	if _, ok := os.LookupEnv("STORE_FILE"); ok {
		storeFileMetrics = cfgENV.StoreFile
	} else if *storeFilePtr != "" {
		storeFileMetrics = *storeFilePtr
	} else if jsonCfg.StoreFile != "" {
		storeFileMetrics = jsonCfg.StoreFile
	} else {
		storeFileMetrics = constants.StoreFile
	}

	keyHash := cfgENV.Key
	if _, ok := os.LookupEnv("KEY"); !ok {
		keyHash = *keyFlag
	}

	var databaseDsn string
	if _, ok := os.LookupEnv("DATABASE_DSN"); ok {
		databaseDsn = cfgENV.DatabaseDsn
	} else if *keyDatabaseDsn != "" {
		databaseDsn = *keyDatabaseDsn
	} else {
		databaseDsn = jsonCfg.StoreFile
	}

	var patchCryptoKey string
	if _, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		patchCryptoKey = cfgENV.CryptoKey
	} else if *cryptoKeyFlag != "" {
		patchCryptoKey = *cryptoKeyFlag
	} else {
		patchCryptoKey = jsonCfg.CryptoKey
	}

	MapTypeStore := make(repository.MapTypeStore)

	if databaseDsn != "" {
		typeDB := repository.TypeStoreDataDB{}
		MapTypeStore[constants.MetricsStorageDB.String()] = &typeDB
	} else if storeFileMetrics != "" {
		typeFile := repository.TypeStoreDataFile{}
		MapTypeStore[constants.MetricsStorageFile.String()] = &typeFile
	}

	constants.Logger.Log = zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

	return ServerConfig{
		StoreInterval:      storeIntervalMetrics,
		StoreFile:          storeFileMetrics,
		Restore:            restoreMetric,
		Address:            addressServ,
		Key:                keyHash,
		DatabaseDsn:        databaseDsn,
		TypeMetricsStorage: MapTypeStore,
		CryptoKey:          patchCryptoKey,
	}, nil
}
