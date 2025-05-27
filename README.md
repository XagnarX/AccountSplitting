```bash

go run main.go genwallet -n 5 -o my.csv -d ./mydir
go run main.go genmnemonic -n 2 -o m.csv   # 默认在 ./wallets 下


go run main.go verifycsv -f mydir/my.csv
go run main.go verifycsv -f wallets/m.csv
```