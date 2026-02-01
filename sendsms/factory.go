package sendsms

import (
	"fmt"
)

// 支持的服务商列表
const (
	ProviderAliyun     = "aliyun"
	ProviderTencent    = "tencent"
	ProviderBaidu      = "baidu"
	ProviderHuawei     = "huawei"
	ProviderNetease    = "netease"
	ProviderRongerlian = "ronglian"
	ProviderAurora     = "aurora"
	ProviderChuanglan  = "chuanglan"
	ProviderTwilio     = "twilio"
)

// NewProvider 根据配置创建服务商实例
func NewProvider(providerType string, config interface{}) (SMSProvider, error) {
	switch providerType {
	case ProviderAliyun:
		// TODO: 实现阿里云服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderTencent:
		// TODO: 实现腾讯云服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderBaidu:
		// TODO: 实现百度云服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderHuawei:
		// TODO: 实现华为云服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderNetease:
		// TODO: 实现网易云信服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderRongerlian:
		// TODO: 实现容联云服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderAurora:
		// TODO: 实现极光服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderChuanglan:
		// TODO: 实现创蓝253服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	case ProviderTwilio:
		// TODO: 实现Twilio服务商
		return nil, fmt.Errorf("provider %s not implemented yet", providerType)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// NewProviders 批量创建服务商实例
func NewProviders(providerNames []string, config interface{}) (map[string]SMSProvider, error) {
	providers := make(map[string]SMSProvider)

	for _, name := range providerNames {
		provider, err := NewProvider(name, config)
		if err != nil {
			return nil, fmt.Errorf("create provider %s failed: %w", name, err)
		}
		providers[name] = provider
	}

	return providers, nil
}
