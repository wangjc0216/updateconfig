package main

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"log"
)

var (
	orghubName = "CrosshubMSP"
)

func main_invokecc() {
	sdk1, err := fabsdk.New(config.FromFile(org1CfgPath))
	if err != nil {
		log.Println(err)
		return
	}
	rcp := sdk1.Context(fabsdk.WithOrg(orghubName), fabsdk.WithUser("admin"))

	rc1, err := resmgmt.New(rcp)
	if err != nil {
		log.Panicf("failed to create resource client: %s", err)
		return
	}
	log.Println(rc1)
}
