package service

// ExportGenerateOTP exposes the private generateOTP for unit tests.
func ExportGenerateOTP(length int) (string, error) {
	return generateOTP(length)
}
