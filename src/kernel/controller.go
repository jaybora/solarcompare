package kernel

import (
	"web"
)


type controller struct {
	live map[uint16]chan chan web.PvData
}
