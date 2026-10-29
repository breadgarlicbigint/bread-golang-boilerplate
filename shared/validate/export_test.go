package validate

// FallbackMessageForTest exposes the no-translator English renderer to the
// package's external test, which asserts it stays in lockstep with the
// `validation` block of locales/en.json.
func FallbackMessageForTest(key string, data map[string]string) string {
	return fallbackMessage(key, data)
}
