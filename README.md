```bash

go run main.go genwallet -n 5 -o my.csv -d ./mydir
go run main.go genmnemonic -n 2 -o m.csv   # 默认在 ./wallets 下
# 生成钱包的命令
go run main.go genmnemonic -n 1000 -o k2.csv -d wallets/S

# 验证钱包私钥是有准确的命令
go run main.go verifycsv -f wallets/m.csv
```


## 分账合约地址
```bash
0x61e0336Ba3bEd95deD28b01ef9cD015d7F32437d
0xe17b3422c8c172C0B8Ee434e33DA12aFf3A211B2
0xCf33569587c5d6Dc7A3c42fc1653334E41Ac94db
0x95A14Eeb790e7E7AFFA7972Ad1dfd72e89DB777F
0x00C8F12f9220Be9b43830123543d364154D0b0b5
```

```bash

go run main.go batch-transfer --csv "wallets/S/1w.csv" --amount 0.0000000121 --max-wallets 3
go run main.go batch-transfer --csv "wallets/S/k2.csv" --amount 0.00023 --rpc https://bsc-dataseed2.binance.org/

go run main.go single-transfer --csv "wallets/S/k2.csv" --delay 2 --rpc https://bsc-dataseed2.binance.org/
go run main.go single-transfer --csv "wallets/S/1w.csv" --target "0x123..." --amount 0.0001 --max-wallets 3 --delay 10


go run main.go batch-transfer --csv "wallets/S/k6.csv" --amount 0.00023 --rpc https://bsc-dataseed1.defibit.io/
go run main.go single-transfer --csv "wallets/S/k6.csv" --delay 2 --rpc https://bsc-dataseed1.defibit.io/

go run main.go batch-transfer --csv "wallets/S/k7.csv" --amount 0.00023 --rpc https://bsc-dataseed2.binance.org/
go run main.go single-transfer --csv "wallets/S/k7.csv" --delay 2 --rpc https://bsc-dataseed2.binance.org/

go run main.go batch-transfer --csv "wallets/S/k8.csv" --amount 0.00023 --rpc https://bsc-dataseed3.ninicoin.io/
go run main.go single-transfer --csv "wallets/S/k8.csv" --delay 2 --rpc https://bsc-dataseed3.ninicoin.io/

# 使用默认rpc分账
# --sender-csv 指定分账的钱包私钥csv文件 默认：wallets/senders/w1.csv
# --sender-index 指定分账的钱包index 默认：0（第一个）
go run main.go batch-transfer --csv "wallets/S/k5.csv" --amount 0.00023
# 使用默认rpc转账0.0001BNB 到 0x774d0d4281217deDB7ae7797D69968D6Ea07c1Ae
go run main.go single-transfer --csv "wallets/S/k5.csv" --delay 2
```



