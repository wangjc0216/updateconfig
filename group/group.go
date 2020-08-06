package group

import (
	"bytes"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/resource"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/common/tools/configtxgen/encoder"
	genesisconfig "github.com/hyperledger/fabric/common/tools/configtxgen/localconfig"
	"github.com/hyperledger/fabric/common/tools/configtxlator/update"
	"github.com/hyperledger/fabric/protos/common"
	"log"
	"os"
)

type Client struct {
	sdk *fabsdk.FabricSDK
	ctx fab.ClientContext //Client supplies the configuration and signing identity to client objects
	rcp context.ClientProvider
	rc  *resmgmt.Client
	g   *Group
}

//func (c *Client) newFabricSDK(cfgPath string) (err error) {
//	c.sdk, err = fabsdk.New(config.FromFile(cfgPath))
//	if err != nil {
//		log.Println("failed to create fabric sdk: ", err)
//		return
//	}
//	c.rcp = c.sdk.Context(fabsdk.WithUser(c.g.User), fabsdk.WithOrg(c.g.OrgName))
//	c.ctx, err = c.rcp()
//	if err != nil {
//		log.Println("c.ctx err", err)
//		return
//	}
//	c.rc, err = resmgmt.New(c.rcp)
//	if err != nil {
//		log.Println("resgment new rcp err")
//		return
//	}
//	return
//}

func (c *Client) queryBlockFromOrderer(channelid string) (*common.Block, error) {
	originalConfigBlock, err := c.rc.QueryConfigBlockFromOrderer(channelid)
	if err != nil {
		log.Println("rc queryconfigblock err: ", err)
		return nil, err
	}
	return originalConfigBlock, nil
}

type Group struct {
	TxPath  string //configtx.yaml所在路径
	SdkPath string //sdk的配置文件路径
	OrgName string
	User    string //amdin,client,peer
	//CfgGroup *common.ConfigGroup
	*Client
}

func (g *Group) newClient(cfgPath string) (*Client, error) {
	c := new(Client)
	c.g = g
	var err error
	c.sdk, err = fabsdk.New(config.FromFile(cfgPath))
	if err != nil {
		log.Println("failed to create fabric sdk: ", err)
		return nil, err
	}
	c.rcp = c.sdk.Context(fabsdk.WithUser(c.g.User), fabsdk.WithOrg(c.g.OrgName))
	c.ctx, err = c.rcp()
	if err != nil {
		log.Println("c.ctx err", err)
		return nil, err
	}
	c.rc, err = resmgmt.New(c.rcp)
	if err != nil {
		log.Println("resgment new rcp err")
		return nil, err
	}
	return c, nil
}

type GroupOptions func(*Group)

func WithTxPath(path string) func(*Group) {
	return func(g *Group) {
		g.TxPath = path
	}
}
func WithUserType(user string) func(*Group) {
	return func(g *Group) {
		if user != "admin" && user != "peer" && user != "client" {
			g.User = "admin"
		} else {
			g.User = user
		}
	}
}
func WithSDKPath(path string) func(*Group) {
	return func(g *Group) {
		g.SdkPath = path
	}
}
func NewCfgGroup(orgName string, opts ...GroupOptions) (*Group, error) {
	var err error
	g := &Group{
		TxPath:  "./configtxdir/",
		OrgName: orgName, //"CrosshubMSP",
		User:    "admin",
		//Client:  new(Client), //todo 初始化的方式好吗
	}
	for _, opt := range opts {
		opt(g)
	}
	_, err = os.Stat(g.SdkPath)
	//
	if err == nil {
		g.Client, err = g.newClient(g.SdkPath)
		if err != nil {
			return nil, err
		}
	}
	return g, nil
}

//从config.tx文件实例化configGroup对象
func (g *Group) GenCfgGroupFromTx() (*common.ConfigGroup, error) {
	if g.TxPath == "" {
		return nil, errors.New("txpath shoud not be empty")
	}
	var cfgG *common.ConfigGroup
	var topLevelConfig *genesisconfig.TopLevel
	topLevelConfig = genesisconfig.LoadTopLevel(g.TxPath)
	var err error
	for _, org := range topLevelConfig.Organizations {
		if org.Name == g.OrgName {
			cfgG, err = encoder.NewConsortiumOrgGroup(org)
			if err != nil {
				log.Println("bad org definition for org :", err)
				return nil, err
			}
			return cfgG, nil
		}
	}
	return nil, errors.New("org gen ConfigGroup fail")
}
func (g *Group) GetChannelConfig(channelid string) (*common.Config, error) {
	block, err := g.Client.queryBlockFromOrderer(channelid)
	if err != nil {
		return nil, err
	}
	config, err := resource.ExtractConfigFromBlock(block)
	if err != nil {
		log.Println("extractConfigFromBlock err:", err)
		return nil, err
	}
	return config, nil
}
func (g *Group) CreateConfigSignature(envelop []byte) (*common.ConfigSignature, error) {
	return g.rc.CreateConfigSignatureFromReader(g.ctx, bytes.NewReader(envelop))
}
func (g *Group) SaveChannel(channelid string, channelCfg []byte, sigs ...*common.ConfigSignature) error {
	resp, err := g.rc.SaveChannel(resmgmt.SaveChannelRequest{
		ChannelID:     channelid,
		ChannelConfig: bytes.NewBuffer(channelCfg)}, resmgmt.WithConfigSignatures(sigs...))
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("add/update success!,resp txid is ", resp.TransactionID)
	return nil
}

func GetCompute(originalCfg, updateCfg *common.Config, channelid string) (*common.ConfigUpdate, error) {
	computeUpdate, err := update.Compute(originalCfg, updateCfg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	computeUpdate.ChannelId = channelid
	return computeUpdate, nil
}

//将更新配置扩展成Envelope，以便发送给fabric网络
func AssembleEnvelop(computeUpdate *common.ConfigUpdate, channelid string) ([]byte, error) {
	// 组装channelHeader
	chHeader := new(common.ChannelHeader)
	chHeader.Type = 2
	chHeader.ChannelId = channelid
	//组装ConfigUpdateEnvelope
	cfgupdateBytes, err := proto.Marshal(computeUpdate)
	if err != nil {
	}
	cfgEnv := new(common.ConfigUpdateEnvelope)
	cfgEnv.ConfigUpdate = cfgupdateBytes
	//组装payload
	cfgEnvBytes, err := proto.Marshal(cfgEnv)
	if err != nil {
	}
	chHeaderBytes, err := proto.Marshal(chHeader)
	if err != nil {
	}
	payload := new(common.Payload)
	payload.Header = new(common.Header)
	payload.Data = cfgEnvBytes
	payload.Header.ChannelHeader = chHeaderBytes
	//组装envelope
	payloadBytes, err := proto.Marshal(payload)
	if err != nil {
	}
	envelope := new(common.Envelope)
	envelope.Payload = payloadBytes
	//将序列化的envelope返回
	return proto.Marshal(envelope)
}

//Create 第一次创建的意思，Gen，生成、导出，以一种形式转化为另一种形式
