package main

import (
	"github.com/wangjc/updateconfig/org"
	"testing"
)

var (
	txpath      = "./configtxdir/"
	MyChannel   = "mychannel"
	org1CfgPath = "./sdkconfig/org1sdk-config.yaml"
	org2CfgPath = "./sdkconfig/org2sdk-config.yaml"
	org1Name    = "Org1"
	org2Name    = "Org2"
	crossName   = "CrosshubMSP"
)

//应用通道增加组织
func TestADDORG(t *testing.T) {
	crossG, err := org.NewCfgGroup(crossName, org.WithTxPath(txpath))
	if err != nil {
		t.Fatal(err)
	}
	org1G, err := org.NewCfgGroup(org1Name, org.WithSDKPath(org1CfgPath))
	if err != nil {
		t.Fatal(err)
	}
	latestedCfg1, err := org1G.GetChannelConfig(MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	crossCfgGroup, err := crossG.GenCfgGroupFromTx()
	if err != nil {
		t.Fatal(err)
	}
	latestedCfg2, err := org1G.GetChannelConfig(MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	latestedCfg2.ChannelGroup.Groups["Application"].Groups["CrosshubMSP"] = crossCfgGroup

	cfgUpdate, err := org.GetCompute(latestedCfg1, latestedCfg2, MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	envBytes, err := org.AssembleEnvelop(cfgUpdate, MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	sig1, err := org1G.CreateConfigSignature(envBytes)
	if err != nil {
		t.Fatal(err)
	}
	org2G, err := org.NewCfgGroup(org2Name, org.WithSDKPath(org2CfgPath))
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := org2G.CreateConfigSignature(envBytes)
	if err != nil {
		t.Fatal(err)
	}
	err = org1G.SaveChannel(MyChannel, envBytes, sig1, sig2)
	if err != nil {
		t.Fatal(err)
	}
}
