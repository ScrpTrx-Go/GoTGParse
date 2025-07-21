package analyzer

func GetRegionKeys(regionsMap map[string]string) []string {
	regions := make([]string, 0, len(regionsMap))
	for region := range regionsMap {
		regions = append(regions, region)
	}
	return regions
}
