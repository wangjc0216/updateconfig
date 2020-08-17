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
	//实例化"第三家"组织，传入交易的路径
	crossG, err := org.NewCfgGroup(crossName, org.WithTxPath(txpath))
	if err != nil {
		t.Fatal(err)
	}
	//实例化Org1
	org1G, err := org.NewCfgGroup(org1Name, org.WithSDKPath(org1CfgPath))
	if err != nil {
		t.Fatal(err)
	}
	//获取mychannel中最新配置
	latestedCfg1, err := org1G.GetChannelConfig(MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	//根据配置文件，生成第三家组织的配置
	crossCfgGroup, err := crossG.GenCfgGroupFromTx()
	if err != nil {
		t.Fatal(err)
	}
	// 获取mychannel中最新配置，这里重复获取两次是为了接下来的比对
	latestedCfg2, err := org1G.GetChannelConfig(MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	//组装最新的通道配置
	latestedCfg2.ChannelGroup.Groups["Application"].Groups["CrosshubMSP"] = crossCfgGroup

	cfgUpdate, err := org.GetCompute(latestedCfg1, latestedCfg2, MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	envBytes, err := org.AssembleEnvelop(cfgUpdate, MyChannel)
	if err != nil {
		t.Fatal(err)
	}
	//Org1,Org2 为最新通道配置签名
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
	//两个签名传给Org1，由Org1来发起更新通道配置的交易
	err = org1G.SaveChannel(MyChannel, envBytes, sig1, sig2)
	if err != nil {
		t.Fatal(err)
	}
}
