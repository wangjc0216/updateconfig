module github.com/wangjc/updateconfig

go 1.13

require (
	github.com/Shopify/sarama v1.26.4 // indirect
	github.com/fsouza/go-dockerclient v1.6.5 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/hashicorp/go-version v1.2.1 // indirect
	github.com/hyperledger/fabric v1.4.3
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a // indirect
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta2
)

//replace github.com/hyperledger/fabric-protos-go v0.0.0-20200707132912-fee30f3ccd23 => /Users/jcwang/workdir/basefabricteam/updateconfig/vendor/github.com/hyperledger/fabric-protos-go
//replace github.com/hyperledger/fabric-sdk-go v1.0.0-beta2 => /Users/jcwang/workdir/basefabricteam/updateconfig/vendor/github.com/hyperledger/fabric-sdk-go
replace github.com/hyperledger/fabric-sdk-go v1.0.0-beta2 => ./vendor/github.com/hyperledger/fabric-sdk-go
