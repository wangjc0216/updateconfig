# 更新Fabric网络通道配置demo

## 概述
通过fabric与fabric-sdk的调用对Fabric网络进行更新，因为fabric-sdk引用的proto包和fabric引用的包有一定出入，重复引用会产生类似如下
报错，所以在使用将fabric-sdk-go包重新修改放到vendor中用**go mod replace**来替换。
```bash
...
    /usr/lib/go/src/runtime/panic.go:464 +0x3e6
github.com/golang/protobuf/proto.RegisterEnum(0xbbca60, 0x29, 0xc820199980, 0xc8201999b0)
    /path/to/third_party/go/ptypes/src/github.com/golang/protobuf/proto/properties.go:811 +0xe2
third_party/proto/google/protobuf.init.4()
    third_party/proto/google/protobuf/descriptor.pb.go:1913 +0x706
third_party/proto/google/protobuf.init()
    third_party/proto/google/protobuf/descriptor.pb.go:2798 +0x5cd
main.init()
...

```


## 功能列表

### 1.【更新通道配置】动态在应用通道添加新的组织

我们**首先启动fabric-sample中的byfn网络**，当前网络联盟中有两家组织Org1，Org2。

大致流程如下：

1.和官方文档中的例子相同，我们首先根据configtx.yaml生成新组织配置信息（**common.ConfigGroup**）。

2.获取通道最新的通道配置，手动将新组织信息加到通道配置中，并进行比较，得到更新信息：**common.ConfigUpdate**

3.单单通过ConfigUpdate是不能更新通道的，我们需要将更新信息封装到指定格式中，也就是信息**common.Envelope**

4.更新通道配置需要通道内所有组织都进行签名，我们就需要使用Org1和Org2的管理员sk来对这个Envelope签名

5.在这个网络中我们得到org1和org2的签名sig1和sig2，将Envelope和这两个签名通过Org1或者Org2发起更新通道配置交易即可更新成功。

### 2.【更新通道配置】动态删除通道内组织

与例子1流程类似。删除了byfn网络mychannel通道中Org2组织


## 本地使用

本demo基于byfn开发，运行程序前先启动byfn网络，同时将sdkconfig目录下sdk配置文件具体证书目录进行修改，与本地一致。
