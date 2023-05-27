package libxray

import (
	"github.com/xtls/xray-core/common/uuid"
)

func CustomUUID(str string) string {
	id, err := uuid.ParseString(str)
	if err != nil {
		return err.Error()
	}
	return id.String()
}
