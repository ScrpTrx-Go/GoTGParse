package config

type TDLibConfig struct {
	UseTestDc           bool       `yaml:"use_test_dc"`
	DatabaseDirectory   string     `yaml:"database_directory"`
	FilesDirectory      string     `yaml:"files_directory"`
	UseFileDatabase     bool       `yaml:"use_file_database"`
	UseChatInfoDatabase bool       `yaml:"use_chat_info_database"`
	UseMessageDatabase  bool       `yaml:"use_message_database"`
	UseSecretChats      bool       `yaml:"use_secret_chats"`
	APIID               int32      `yaml:"api_id"`
	APIHash             string     `yaml:"api_hash"`
	SystemLanguageCode  string     `yaml:"system_language_code"`
	DeviceModel         string     `yaml:"device_model"`
	SystemVersion       string     `yaml:"system_version"`
	ApplicationVersion  string     `yaml:"application_version"`
	LogLevel            int        `yaml:"log_level"`
	Usernames           []string   `yaml:"usernames"`
	GetHistory          GetHistory `yaml:"gethistory"`
}

type GetHistory struct {
	FromMessageID int64 `yaml:"from_message_id"`
	Offset        int32 `yaml:"offset"`
	Limit         int32 `yaml:"limit"`
	OnlyLocal     bool  `yaml:"only_local"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type LoggerConfig struct {
	Level      string `yaml:"level"`
	FilePath   string `yaml:"file_path"`
	Production bool   `yaml:"production"`
}
