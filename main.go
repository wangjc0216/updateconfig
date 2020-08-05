package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/resource"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	"github.com/hyperledger/fabric/common/tools/protolator"
	"github.com/hyperledger/fabric/protos/common"
	"log"
)

//1.根据configtx.yaml生成配置文件json
//等同于../../bin/configtxgen -printOrg Org3MSP > ../channel-artifacts/org3.json
func GenerateCfgStr(txconfigPath string) (string, *common.ConfigGroup, error) {
	var topLevelConfig *genesisconfig.TopLevel
	printOrg := "CrosshubMSP"
	buf := bytes.NewBuffer(make([]byte, 1024))

	if txconfigPath != "" {
		topLevelConfig = genesisconfig.LoadTopLevel(txconfigPath)
	} else {
		topLevelConfig = genesisconfig.LoadTopLevel()
	}
	for _, org := range topLevelConfig.Organizations {
		if org.Name == printOrg {
			og, err := encoder.NewConsortiumOrgGroup(org)
			if err != nil {
				log.Println("bad org definition for org :", err)
				return "", nil, err
			}
			if err := protolator.DeepMarshalJSON(buf, og); err != nil {
				log.Println("malformed org definition for org: ", err, org.Name)
				return "", nil, err
			}
			return buf.String(), og, nil
		}
	}
	return "", nil, nil
}

var (
	txconfigPath = "./configtxdir/"
	org1CfgPath  = "./sdkconfig/org1sdk-config.yaml"
	org2CfgPath  = "./sdkconfig/org2sdk-config.yaml"
	org1Name     = "Org1"
	org2Name     = "Org2"
	orgAdmin     = "Admin"
	channelID    = "mychannel"
)

func main() {
	_, crossCfg, err := GenerateCfgStr(txconfigPath)
	if err != nil {
		log.Println(err)
		return
	}

	sdk1, err := fabsdk.New(config.FromFile(org1CfgPath))
	if err != nil {
		log.Panicf("failed to create fabric sdk: %s", err)
	}

	sdk2, err := fabsdk.New(config.FromFile(org2CfgPath))
	if err != nil {
		log.Panicf("failed to create fabric sdk: %s", err)
	}

	originalCfg, err := GetLastestBlock(sdk1, org1Name, orgAdmin, channelID)
	if err != nil {
		log.Println(err)
		return
	}

	updateCfg, err := GetLastestBlock(sdk1, org1Name, orgAdmin, channelID)
	if err != nil {
		log.Println(err)
		return
	}
	updateCfg.ChannelGroup.Groups["Application"].Groups["CrosshubMSP"] = crossCfg

	computeUpdate, err := update.Compute(originalCfg, updateCfg)
	if err != nil {
		log.Println(err)
		return
	}
	computeUpdate.ChannelId = channelID //todo 不知道为啥，和增加org3 的脚本比对没有对应上
	//更新数据读写集打印
	diffBuf := bytes.NewBuffer(make([]byte, 0))
	err = protolator.DeepMarshalJSON(diffBuf, computeUpdate)
	if err != nil {
		log.Println(err)
		return
	}
	computeUpdateMap := make(map[string]interface{})
	err = json.Unmarshal(diffBuf.Bytes(), &computeUpdateMap)
	if err != nil {
		log.Println(err)
		return
	}

	var envelop struct {
		Payload struct {
			Header struct {
				ChannelHeader struct {
					Channelid string `json:"channel_id"`
					Type      int    `json:"type"`
				} `json:"channel_header"`
			} `json:"header"`
			Data struct {
				ConfigUpdate interface{} `json:"config_update"`
			} `json:"data"`
		} `json:"payload"`
	}
	envelop.Payload.Data.ConfigUpdate = computeUpdateMap
	envelop.Payload.Header.ChannelHeader.Type = 2
	envelop.Payload.Header.ChannelHeader.Channelid = "mychannel"

	envelopBs, err := json.Marshal(envelop)
	if err != nil {
		log.Println(err)
		return
	}

	rcp1 := sdk1.Context(fabsdk.WithUser(orgAdmin), fabsdk.WithOrg(org1Name))
	rc1, err := resmgmt.New(rcp1)
	if err != nil {
		log.Panicf("failed to create resource client: %s", err)
		return
	}
	rcp2 := sdk2.Context(fabsdk.WithUser(orgAdmin), fabsdk.WithOrg(org2Name))
	rc2, err := resmgmt.New(rcp2)
	if err != nil {
		log.Panicf("failed to create resource client: %s", err)
		return
	}
	envelopBuffer := bytes.NewBuffer(envelopBs)
	protoEnvelop := new(common.Envelope)
	err = protolator.DeepUnmarshalJSON(envelopBuffer, protoEnvelop)
	if err != nil {
		log.Println(err)
		return
	}
	envelopProto, err := proto.Marshal(protoEnvelop)
	if err != nil {
		log.Println(err)
		return
	}

	ctx1, err := sdk1.Context(fabsdk.WithUser(orgAdmin), fabsdk.WithOrg(org1Name))()
	ctx2, err := sdk2.Context(fabsdk.WithUser(orgAdmin), fabsdk.WithOrg(org2Name))()

	sdk1Sig, err := rc1.CreateConfigSignatureFromReader(ctx1, bytes.NewBuffer(envelopProto))
	if err != nil {
		log.Println(err)
		return
	}
	sdk2Sig, err := rc2.CreateConfigSignatureFromReader(ctx2, bytes.NewBuffer(envelopProto))
	if err != nil {
		log.Println(err)
		return
	}

	//cfgUpdateEnv.Signatures = append(cfgUpdateEnv.Signatures, sdk1Sig, sdk2Sig)

	saveReq := resmgmt.SaveChannelRequest{
		ChannelID:         channelID,
		ChannelConfig:     bytes.NewBuffer(envelopProto),     // 是envelop的反序列化
		SigningIdentities: []msp.SigningIdentity{ctx1, ctx2}, //需要在MSP中增加identity
	}

	saveResp, err := rc1.SaveChannel(saveReq, resmgmt.WithConfigSignatures(sdk1Sig, sdk2Sig))
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(saveResp.TransactionID)

}

//2.获取最新配置块
func GetLastestBlock(sdk *fabsdk.FabricSDK, orgName, orgAdmin, channelID string) (*common.Config, error) {

	rcp := sdk.Context(fabsdk.WithUser(orgAdmin), fabsdk.WithOrg(orgName))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		log.Panicf("failed to create resource client: %s", err)
	}
	log.Println("Initialized resource client")

	originalConfigBlock, err := rc.QueryConfigBlockFromOrderer(channelID)
	if err != nil {
		log.Println("rc queryconfigblock err: ", err)
		return nil, err
	}
	originalConfig, err := resource.ExtractConfigFromBlock(originalConfigBlock)
	if err != nil {
		log.Println("extractConfigFromBlock err:", err)
		return nil, err
	}
	return originalConfig, nil

}
