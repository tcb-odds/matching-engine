package info

import (
	"fmt"
)

func PrintAppInfo(appName string, appVersion string) {
	fmt.Printf(">>> %s v%s <<<\n", appName, appVersion)
}
