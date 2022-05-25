package main

import (
	"flag"
	"fmt"

	"wallet-srv/lib/pkg/subkey/sr25519"

	"wallet-srv/lib/pkg/subkey"
)

func main() {
	s := flag.String("secret", "", "Secret key in Hex")
	m := flag.String("msg", "", "Message to be signed in Hex")
	flag.Parse()

	msg, ok := subkey.DecodeHex(*m)
	if !ok {
		panic(fmt.Errorf("invalid hex"))
	}

	kr, err := subkey.DeriveKeyPair(sr25519.Scheme{}, *s)
	if err != nil {
		panic(err)
	}

	sig, err := kr.Sign(msg)
	if err != nil {
		panic(err)
	}

	fmt.Println(kr.Verify(msg, sig))
}
