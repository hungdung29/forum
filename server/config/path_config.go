package config

// BasePath is used for templates and database paths
// Empty string "" works when running from repository root: go run ./cmd
// Set to "../" when running from cmd directory: cd cmd && go run .
// Or set via BASE_PATH environment variable
var BasePath = ""
