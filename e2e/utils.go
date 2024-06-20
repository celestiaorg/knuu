package e2e

import (
	"fmt"
	"time"
)

func DefaultTestScope() string {
	t := time.Now()
	return fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
}
