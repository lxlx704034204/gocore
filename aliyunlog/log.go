package aliyunlog

import (
	"errors"
	"fmt"
	"os"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/aliyun/aliyun-log-go-sdk/producer"
	"github.com/sunmi-OS/gocore/viper"
	"github.com/tidwall/gjson"
)

// AliyunLog 阿里云日志配置结构体
type AliyunLog struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Project   string
	LogStore  string
	HostName  string
	Log       *producer.Producer
}

// LogClient 对外原生实例
var LogClient AliyunLog

// InitLog 初始化日志
func InitLog(configName, LogStore string) {
	hostname, _ := os.Hostname()
	LogClient = AliyunLog{
		Project:   viper.GetEnvConfig(configName + ".Project"),
		Endpoint:  viper.GetEnvConfig(configName + ".Endpoint"),
		AccessKey: viper.GetEnvConfig(configName + ".AccessKey"),
		SecretKey: viper.GetEnvConfig(configName + ".SecretKey"),
		LogStore:  LogStore,
		HostName:  hostname,
	}
	err := checkConfig(LogClient)
	if err != nil {
		panic(err)
	}

	fmt.Println("test")

	producerConfig := producer.GetDefaultProducerConfig()
	producerConfig.Endpoint = LogClient.Endpoint
	producerConfig.AccessKeyID = LogClient.AccessKey
	producerConfig.AccessKeySecret = LogClient.SecretKey
	LogClient.Log = producer.InitProducer(producerConfig)

	LogClient.Log.Start()

	logmsg := producer.GenerateLog(uint32(time.Now().Unix()), map[string]string{"content": "log-start"})
	err = LogClient.Log.SendLog(LogClient.Project, LogClient.LogStore, "start", LogClient.HostName, logmsg)
	if err != nil {
		panic(err)
	}
}

// Info 记录info日志
func Info(topic string, logs map[string]string) error {
	logs["level"] = "info"
	logmsg := producer.GenerateLog(uint32(time.Now().Unix()), logs)
	return LogClient.Log.SendLog(LogClient.Project, LogClient.LogStore, topic, LogClient.HostName, logmsg)
}

// Error 记录异常日志
func Error(topic string, logs map[string]string) error {
	logs["level"] = "error"
	logmsg := producer.GenerateLog(uint32(time.Now().Unix()), logs)
	return LogClient.Log.SendLog(LogClient.Project, LogClient.LogStore, topic, LogClient.HostName, logmsg)
}

// Close 关闭日志服务
func Close() {
	if LogClient.Log != nil {
		LogClient.Log.SafeClose()
	}
}

// checkConfig 验证配置是否缺少 自动创建LogStore
func checkConfig(conf AliyunLog) (err error) {
	if conf.AccessKey == "" || conf.Endpoint == "" || conf.Project == "" || conf.LogStore == "" || conf.SecretKey == "" {
		err = errors.New("config Missing parameter")
	}

	// 创建 LogStore 默认存储30天，2个分片自动扩容最大64片
	Client := sls.CreateNormalInterface(conf.Endpoint, conf.AccessKey, conf.SecretKey, "")
	err = Client.CreateLogStore(conf.Project, conf.LogStore, 30, 2, true, 64)
	if err != nil {
		if gjson.Parse(err.Error()).Get("errorCode").String() == "LogStoreAlreadyExist" {
			err = nil
		}
	}

	// 创建索引
	index := sls.Index{
		Keys: map[string]sls.IndexKey{
			"content": {
				Token:         []string{`,`, ` `, `'`, `"`, `;`, `=`, `(`, `)`, `[`, `]`, `{`, `}`, `?`, `@`, `&`, `<`, `>`, `/`, `:`, `\n`, `\t`, `\r`},
				CaseSensitive: false,
				Type:          "text",
				Chn:           true,
				DocValue:      true,
			},
		},
		Line: &sls.IndexLine{
			Token:         []string{`,`, ` `, `'`, `"`, `;`, `=`, `(`, `)`, `[`, `]`, `{`, `}`, `?`, `@`, `&`, `<`, `>`, `/`, `:`, `\n`, `\t`, `\r`},
			CaseSensitive: false,
			IncludeKeys:   []string{},
			ExcludeKeys:   []string{},
			Chn:           true,
		},
	}
	err = Client.CreateIndex(conf.Project, conf.LogStore, index)
	if err != nil {
		if gjson.Parse(err.Error()).Get("errorCode").String() == "IndexAlreadyExist" {
			err = nil
		}
	}

	Client.Close()
	return
}
