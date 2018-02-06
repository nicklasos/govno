# GO VNO
VNO protocol implementation in Go<br>
VNO is stands for Very Needed Object<br><br>

Example of govno.toml
```toml
[[database]]
name = "database_name" # database name
host = "127.0.0.1"

    [[database.vno]]
    name = "daily"
    path = "{month}/{day}/file.sql.gz"

    [[database.vno]]
    name = "monthly"
    path = "{year}/{month}/{day}/file.sql.gz"

[[database]]
name = "another_database"
host = "8.8.8.8"

    [[database.vno]]
    name = "daily"
    path = "{host}/mysql/daily/{year}/{month}/{day}.sql.gz"

```

Example of crontab file
```
0 22 * * * govno daily >> /dev/null 2>&1
```