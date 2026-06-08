// Package version содержит информацию о версии и дате сборки клиента GophKeeper.
package version

import "fmt"

// Version — семантическая версия клиента.
var Version = "dev"

// BuildDate — дата и время сборки бинарного файла клиента.
var BuildDate = "unknown"

// String возвращает строковое представление версии и даты сборки.
func String() string {
	return fmt.Sprintf("GophKeeper client %s (built %s)", Version, BuildDate)
}
