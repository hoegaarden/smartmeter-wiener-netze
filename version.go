package smartmeter

import "fmt"

var (
	buildVersion string = "<unknown>"
	buildDate    string = "<unknown>"
)

func VersionInfo() string {
	return fmt.Sprintf("%s (%s)", buildVersion, buildDate)
}
