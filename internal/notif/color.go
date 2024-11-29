package notif

import "fmt"

type Color uint

const (
	Red   Color = 0xff0000
	Green Color = 0x00ff00
	Blue  Color = 0x0000ff
)

func (c Color) HexString() string {
	return fmt.Sprintf("#%x", c)
}

func (c Color) DecString() string {
	return fmt.Sprintf("%d", c)
}

func (c Color) String() string {
	return c.HexString()
}
