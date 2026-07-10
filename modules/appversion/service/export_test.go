package service

// ExportCompareVersions exposes the private compareVersions for unit tests.
func ExportCompareVersions(a, b string) (int, error) {
	return compareVersions(a, b)
}

// ExportDetermineStatus replicates the Check() decision logic for table-driven tests.
func ExportDetermineStatus(clientVersion, minVersion, currentVersion string, forceUpdate bool) UpdateStatus {
	cmp, err := compareVersions(clientVersion, minVersion)
	if err != nil {
		return UpToDate
	}
	switch {
	case cmp < 0 && forceUpdate:
		return UpdateRequired
	case cmp < 0:
		return UpdateAvailable
	default:
		latestCmp, _ := compareVersions(clientVersion, currentVersion)
		if latestCmp < 0 {
			return UpdateAvailable
		}
		return UpToDate
	}
}
