package pinner

import (
	"fmt"
	"io"

	ipfs "github.com/ipfs/go-ipfs-api"
)

type IPFS struct {
	shell *ipfs.Shell
}

func NewIpfsPinner(ipfsAddr string) IPinner {
	return &IPFS{shell: ipfs.NewShell(ipfsAddr)}
}

func (p *IPFS) Pin(fileName string, file io.Reader) (string, error) {
	hash, err := p.shell.Add(file, ipfs.Pin(true))
	if err != nil {
		return "", fmt.Errorf("add repository metadata to ipfs: %w", err)
	}
	return hash, nil
}
