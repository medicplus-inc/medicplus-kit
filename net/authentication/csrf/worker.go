package csrf

import "time"

type CSRFWorker struct{}

func (c *CSRFToken) Start() {
	for {
		now := time.Now().UTC()

		for k, v := range inMemToken {
			if now.After(v.ExpiredAt) {
				delete(inMemToken, k)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
