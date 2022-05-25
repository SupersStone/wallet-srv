package address

import (
	"errors"
	"strings"

	"wallet-srv/lib/pkg/ada/internal/bech32"
	"wallet-srv/lib/pkg/ada/network"

	"github.com/btcsuite/btcutil/base58"
	"github.com/fxamacker/cbor/v2"
)

var (
	ErrUnsupportedAddress = errors.New("invalid/unsupported address type")
)

type Address interface {
	cbor.Marshaler

	// Bytes returns raw bytes for use in tx_outputs
	Bytes() []byte

	// String returns bech32 encoded human readable string
	String() string

	// NetworkInfo returns NetworkInfo with networks network id and protocol magic
	NetworkInfo() *network.NetworkInfo
}

func NewAddress(raw string) (addr Address, err error) {
	var data []byte

	if strings.HasPrefix(raw, "addr") || strings.HasPrefix(raw, "stake") {
		_, data, err = bech32.Decode(raw)
	} else {
		data = base58.Decode(raw)
	}

	if err != nil {
		return
	}

	header := data[0]
	netId := header & 0x0F

	networks := map[byte]network.NetworkInfo{
		byte(1): *network.MainNet(),
		byte(0): *network.TestNet(),
	}

	switch (header & 0xF0) >> 4 {
	// 1000: byron address
	case 0b1000:
		var byron ByronAddress

		if err := cbor.Unmarshal(data, &byron); err != nil {
			return &ByronAddress{}, err
		}

		return &byron, nil

	// 0000: base address: keyhash28,keyhash28
	// 0001: base address: scripthash28,keyhash28
	// 0010: base address: keyhash28,scripthash28
	// 0011: base address: scripthash28,scripthash28
	case 0b0000, 0b0001, 0b0010, 0b0011:
		baseAddr := BaseAddress{
			Network: networks[netId],
			Payment: *readAddrCred(data, header, 4, 1),
			Stake:   *readAddrCred(data, header, 5, 1+28),
		}
		return &baseAddr, nil

	// 0100: pointer address: keyhash28, 3 variable length uint
	// 0101: pointer address: scripthash28, 3 variable length uint
	case 0b0100, 0b0101:
		var pointerAddr PointerAddress

		return &pointerAddr, nil

	// 0110: enterprise address: keyhash28
	// 0111: enterprise address: scripthash28
	case 0b0110, 0b0111:
		// header + keyhash

		const enterpriseAddrSize = 1 + 28
		if len(data) < enterpriseAddrSize {
			return nil, errors.New("cbor not enough error")
		}
		if len(data) > enterpriseAddrSize {
			return nil, errors.New("cbor trailing data error")
		}
		netw := networks[netId]
		res := NewEnterpriseAddress(&netw, readAddrCred(data, header, 4, 1))
		return res, nil
	case 0b1110, 0b1111:
		const rewardAddrSize = 1 + 28
		if len(data) < rewardAddrSize {
			return nil, errors.New("cbor not enough error")
		}
		if len(data) > rewardAddrSize {
			return nil, errors.New("cbor trailing data error")
		}
		netw := networks[netId]
		res := NewRewardAddress(&netw, readAddrCred(data, header, 4, 1))
		return res, nil

	default:
		return nil, ErrUnsupportedAddress
	}
}

func readAddrCred(data []byte, header byte, bit byte, pos int) *StakeCredential {
	hashBytes := data[pos : pos+28]

	if header&(1<<bit) == 0 {
		return NewKeyStakeCredential(
			hashBytes,
		)
	}
	return NewScriptStakeCredential(
		hashBytes,
	)
}
