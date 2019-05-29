# GO VNO
VNO protocol implementation in Go<br>
VNO is stands for Very Needed Object<br><br>

Example of govno.toml
```toml
[[database]]
name = "database_name" # database name
host = "127.0.0.1"
cnf = "config.cnf"
aws_bucket = "backup-site-test"
aws_id = ""
aws_key = ""
aws_region = "us-west-2"

    [[database.vno]]
    name = "daily"
    path = "{month}/{day}/file.sql.gz"

    [[database.vno]]
    name = "monthly"
    path = "{year}/{month}/{day}/file.sql.gz"

[[database]]
name = "another_database"
host = "8.8.8.8"
...

    [[database.vno]]
    name = "daily"
    path = "{host}/mysql/daily/{year}/{month}/{day}.sql.gz"

```

Example of crontab file<br>
daily - name of vno object
govno.toml - config location
```
0 22 * * * govno daily govno.toml >> /dev/null 2>&1
```

Also you can put your govno.toml to ~/.govno
```
0 22 * * * govno daily >> /dev/null 2>&1
```

config.cnf file
```
[client]
user=root
password=secret
host=localhost
```