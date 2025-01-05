package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/types"
	"golang.org/x/exp/rand"
)

type Clicker struct {
	client      *client.Client
	mainPublic  string
	userAccount types.Account
	minDelay    int
	maxDelay    int
	stopChan    chan struct{}
	running     bool
	mutex       sync.Mutex
	lastGrass   int
}

func (t *Clicker) addDelay() {
	if t.minDelay > 0 || t.maxDelay > 0 {
		delay := t.minDelay
		if t.maxDelay > t.minDelay {
			delay += rand.Intn(t.maxDelay - t.minDelay)
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

func makeInstructionData(discriminator ...int) []byte {
	buf := new(bytes.Buffer)
	for _, val := range discriminator {
		if err := binary.Write(buf, binary.LittleEndian, uint8(val)); err != nil {
			return nil
		}
	}
	return buf.Bytes()
}

func (t *Clicker) getAccountInfo() int {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userAccount := common.PublicKeyFromString(t.mainPublic)
	userInfo, _, _ := common.FindProgramAddress(
		[][]byte{
			[]byte("user"),
			userAccount.Bytes(),
		},
		common.PublicKeyFromString("turboe9kMc3mSR8BosPkVzoHUfn5RVNzZhkrT2hdGxN"),
	)

	accountInfo, err := t.client.GetAccountInfo(ctx, userInfo.ToBase58())
	if err != nil || len(accountInfo.Data) < 16 {
		return t.lastGrass
	}

	grass := int(binary.LittleEndian.Uint64(accountInfo.Data[8:16]))
	t.lastGrass = grass
	return t.lastGrass
}

func (t *Clicker) click() error {
	clickerAccount := t.userAccount
	userAccount := common.PublicKeyFromString(t.mainPublic)
	configAccount, _, _ := common.FindProgramAddress(
		[][]byte{[]byte("configuration")},
		common.PublicKeyFromString("turboe9kMc3mSR8BosPkVzoHUfn5RVNzZhkrT2hdGxN"),
	)
	clickerInfo, _, _ := common.FindProgramAddress(
		[][]byte{
			[]byte("clicker"),
			clickerAccount.PublicKey.Bytes(),
		},
		common.PublicKeyFromString("turboe9kMc3mSR8BosPkVzoHUfn5RVNzZhkrT2hdGxN"),
	)

	userInfo, _, _ := common.FindProgramAddress(
		[][]byte{
			[]byte("user"),
			userAccount.Bytes(),
		},
		common.PublicKeyFromString("turboe9kMc3mSR8BosPkVzoHUfn5RVNzZhkrT2hdGxN"),
	)

	data := makeInstructionData(11, 147, 179, 178, 145, 118, 45, 186, rand.Intn(256))
	if data == nil {
		return fmt.Errorf("构造指令数据失败")
	}

	instruction := types.Instruction{
		ProgramID: common.PublicKeyFromString("turboe9kMc3mSR8BosPkVzoHUfn5RVNzZhkrT2hdGxN"),
		Accounts: []types.AccountMeta{
			{PubKey: clickerInfo, IsSigner: false, IsWritable: false},
			{PubKey: userInfo, IsSigner: false, IsWritable: true},
			{PubKey: configAccount, IsSigner: false, IsWritable: false},
			{PubKey: clickerAccount.PublicKey, IsSigner: true, IsWritable: true},
			{PubKey: common.SysVarInstructionsPubkey, IsSigner: false, IsWritable: false},
		},
		Data: data,
	}

	recentBlockhash, err := t.client.GetLatestBlockhash(context.Background())
	if err != nil {
		return fmt.Errorf("获取区块哈希失败: %w", err)
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        clickerAccount.PublicKey,
			RecentBlockhash: recentBlockhash.Blockhash,
			Instructions:    []types.Instruction{instruction},
		}),
		Signers: []types.Account{clickerAccount},
	})
	if err != nil {
		return fmt.Errorf("构造交易失败: %w", err)
	}

	_, err = t.client.SendTransaction(context.Background(), tx)
	if err != nil {
		if strings.Contains(err.Error(), "InsufficientFundsForRent") {
			return fmt.Errorf("点击次数不足")
		}
		return fmt.Errorf("发送交易失败: %w", err)
	}
	return nil
}

func (t *Clicker) startTask() {
	t.running = true
	for {
		select {
		case <-t.stopChan:
			t.running = false
			return
		default:
			if !t.running {
				return
			}
			if err := t.click(); err != nil {
				time.Sleep(time.Second * 5)
				continue
			}
			t.addDelay()
		}
	}
}

var taskManager = make(map[string]*Clicker)

func main() {
	// 示例：启动多个 Clicker 任务
	clickerConfigs := []struct {
		mainPublic   string
		userPrivate  string
		minDelay     int
		maxDelay     int
	}{
		{"YourMainPublicKey1", "[YourPrivateKey1]", 1000, 2000},
		{"YourMainPublicKey2", "[YourPrivateKey2]", 1000, 2000},
		// 可以继续添加更多配置
	}

	for i, config := range clickerConfigs {
		userPrivateBytes, _ := parsePrivateKey(config.userPrivate)
		userAccount, _ := types.AccountFromBytes(userPrivateBytes)

		clicker := &Clicker{
			client:      client.NewClient("https://eclipse.lgns.net/"),
			mainPublic:  config.mainPublic,
			userAccount: userAccount,
			minDelay:    config.minDelay,
			maxDelay:    config.maxDelay,
			lastGrass:   0,
			stopChan:    make(chan struct{}),
			running:     false,
		}

		taskManager[fmt.Sprintf("task%d", i+1)] = clicker
		go clicker.startTask()
	}

	// 等待任务完成（或添加其他逻辑）
	select {}
}

func parsePrivateKey(keyStr string) ([]byte, error) {
	rawString := strings.Trim(keyStr, "[]")
	numberStrings := strings.Split(rawString, ",")
	var byteArray []byte
	for _, numStr := range numberStrings {
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		byteArray = append(byteArray, byte(num))
	}
	return byteArray, nil
}
