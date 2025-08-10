package ipcoin

type Config struct {
	CloudflareRequired bool   `json:"cloudflareRequired"`
	DBDSN              string `json:"dbDSN"`
	OpenAIAPIKey       string `json:"openaiAPIKey"`
}
