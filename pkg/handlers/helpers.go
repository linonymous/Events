package handlers

import "time"

func (c *Controller) now() time.Time {
	return time.Now().In(c.location)
}
