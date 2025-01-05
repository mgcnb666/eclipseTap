# eclipseTap
安装go
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz

sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

source ~/.profile

go version

进入eclipseTap

cd eclipseTap
打开main.go修改171 172行的内容

{"YourMainPublicKey1", "[YourPrivateKey1]", 1000, 2000},
		{"YourMainPublicKey2", "[YourPrivateKey2]", 1000, 2000},

  替换成你自己的
安装依赖


go mod init eclipseTap

go get github.com/blocto/solana-go-sdk
go get filippo.io/edwards25519
go get github.com/mr-tron/base58
go get golang.org/x/exp/rand
go mod tidy

启动


go run main.go

