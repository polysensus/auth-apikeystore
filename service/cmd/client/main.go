package main

import (
	"encoding/binary"
	"fmt"
	"os"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/robinbryce/apikeystore/apibin"

	// "google.golang.org/grpc"
	"github.com/spf13/pflag"
)

const (
	Name = "apikeyclient"
)

type Config struct {
	Verbose     bool
	GRPC        bool
	HexOutput   bool
	DisplayName string
	Audience    string
	Scopes      string
}

func rawMessage(cfg Config) []byte {

	b := flatbuffers.NewBuilder(0)
	n := b.CreateString(cfg.DisplayName)
	a := b.CreateString(cfg.Audience)
	s := b.CreateString(cfg.Scopes)

	if cfg.Verbose {
		fmt.Printf("n: %s, a %s, s %s\n", cfg.DisplayName, cfg.Audience, cfg.Scopes)
	}

	apibin.CreateRequestStart(b)
	apibin.CreateRequestAddDisplayName(b, n)
	apibin.CreateRequestAddAudience(b, a)
	apibin.CreateRequestAddScopes(b, s)
	b.Finish(apibin.CreateRequestEnd(b))
	return b.FinishedBytes()
}

func main() {

	cfg := Config{}

	f := pflag.NewFlagSet(Name, pflag.ContinueOnError)

	f.BoolVarP(&cfg.Verbose, "verbose", "v", false, "verbosity")
	f.BoolVarP(&cfg.GRPC, "grpc", "g", false, "envelop for grpc transport")
	f.BoolVarP(&cfg.HexOutput, "hex", "x", false, "print output as hex string")
	f.StringVar(&cfg.DisplayName, "displayname", cfg.DisplayName, `display_name`)
	f.StringVar(&cfg.Audience, "audience", cfg.Audience, `audience`)
	f.StringVar(&cfg.Scopes, "scopes", cfg.Scopes, `scopes`)
	pflag.CommandLine.AddFlagSet(f)
	pflag.Parse()

	// var tag uint32
	// tag = (0x1<<3 | 0x10)

	// msgHeader grpc
	// payloadLen 1, sizeLen 4, headerLen = payloadLen + sizeLen
	data := rawMessage(cfg)
	if !cfg.GRPC {
		if cfg.HexOutput {
			fmt.Printf("%x\n", data)
			return
		}
		binary.Write(os.Stdout, binary.BigEndian, data)
		return
	}

	b := make([]byte, 5, len(data)+5)

	// grpc msgHeader does binary.BigEndian.PutUint32(hdr[payloadLen:], uint32(len(data)))
	b[0] = 0x0 // not compressed
	binary.BigEndian.PutUint32(b[1:], uint32(len(data)))
	// binary.BigEndian.PutUint32(b[1+4:], tag)
	b = append(b, data...)
	// fmt.Printf("len:%d\n", len(b))

	if cfg.HexOutput {
		fmt.Printf("%x\n", b)
		return
	}
	binary.Write(os.Stdout, binary.BigEndian, b)
}

// boilerplate from here
func exitOnErr(err error) {
	if err == nil {
		return
	}
	fmt.Printf("error: %v\n", err)
}
