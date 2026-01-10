package config

type GitHubConfig struct {
	Token     string
	Repo      string
	Branch    string
	GameName  string
	SavePath  string
	KeepSaves bool
}