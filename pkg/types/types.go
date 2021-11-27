package types

// just the bits of viper.Viper that we need
type SimpleStringConfig interface {
	GetString(string) string
}
