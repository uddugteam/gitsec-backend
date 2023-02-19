package pinner

import "io"

type IPinner interface {
	Pin(fileName string, file io.Reader) (string, error)
}
