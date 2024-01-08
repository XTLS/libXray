package xray

import (
	"github.com/xtls/xray-core/common/uuid"
)

// convert text to uuid
func CustomUUID(text string) string {
	id, err := uuid.ParseString(text)
	if err != nil {
		return text
	}
	return id.String()
}
