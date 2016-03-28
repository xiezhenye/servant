# servant
A common agent to execute configured command and serve file read write via HTTP protocol

## build
    cd <path to project>
    make
    
By defaults, only mysql database driver are built in. You can use `make DRIVERS="mysql sqlite postgres"` to choose other drivers.

## usage
    servant -conf conf/example.xml >>servant.log &

## config
See conf/example.xml

Servant config file is in xml, may looks like this;

    <?xml version="1.0" encoding="utf-8" ?>
    <config>
        <server>
            <listen>:2465</listen>
            <auth enabled="0">
                <maxTimeDelta>300</maxTimeDelta>
            </auth>
        </server>

        <commands id="db1">
            <command id="foo" runas="mysql" lang="shell">
                <code>echo "hello world $(whoami)"</code>
            </command>
            <command id="grep" lang="exec">
                <code>grep hello</code>
            </command>
            <command id="sleep" timeout="5" lang="exec">
                <code> sleep $t</code>
            </command>
        </commands>
        <files id="db1">
            <dir id="binlog1">
                <root>/data/mysql0</root>
                <allow>get</allow>
                <allow>head</allow>
                <allow>post</allow>
                <allow>delete</allow>
                <allow>put</allow>
                <pattern>log-bin\.\d+</pattern>
        </dir>
        </files>

        <database id="mysql" driver="mysql" dsn="root:@tcp(127.0.0.1:3306)/test">
            <query id="select_1">select 1;</query>
        </database>

        <user id="user1">
            <key>someKey</key>
            <host>192.168.1.0/24</host>
            <files id="db1" />
            <commands id="db1" />
        </user>
    </config>

### `server`

Server level configs. 

#### `server/listen`

Address to bind on and listen, can be `<ip>:<port>` or `:<port>` e.g. `0.0.0.0:2465`, `:2465`

#### `server/auth`

Authorization config. 

Attribute `enabled`:<br />
can be 0 or 1. When authorization disabled, `user` config has no use.

Element `server/auth/maxTimeDelta`:<br />
Max time delta between servant server and client allowed.

### resources group elements

Resources group elements can be `commands`, `files`, `database`, which defines some resource item elements. Each resource group and resource item elements must has an `id` attribute. Client can reference a resource by `/<resource_type>/<group>/<item>`, e.g. `/commands/db1/foo`

### `commands`

Defines a group of commands can be executed, contains some `command` elements.

#### `commands/command`
Attribute `lang`: <br />
Can be `bash`, `exec`. <br />
As `bash`, code is executed as bash, supports if, for, pipe etc, but not supports parameter replacement. <br />
As `exec`, code implies just a single command and arguments separated by spaces. You can use `${param_name}` as a placeholder, and replace it by query parameters. If you want to use an argument contains spaces, you can wrap it by quotes e.g. "hello world", 'hello servant'.

Attribute `runas`:
The system user to execute the command, default is current user. To use `runas`, servant must be run as root.

Attribute `timeout`:<br />
Limit the command execution time in seconds, default is unlimited.

Element `commands/command/code`:<br />
Code of the command to be executed

### `files`

Defines some directories can be accessed.
 
#### `files/dir`

A directory can be accessed.

`files/dir/root`:<br />
The root of the directory. Access will be limited in it.

`files/dir/allow`:<br />
Methods allows to access the files in the directory. can be get, post, put, delete, head. Which means read, create, update, delete, stat. This element can appearances more than one times. 

`files/dir/pattern`:<br />
File name patterns allowed to be access in regular expression. If not defined, all files in the directory can be accessed. This element can appearances more than one times. 

### `database`

A database resource.

Attribute `driver`:<br />
Database driver, supports mysql, sqlite, postgresql. 

Attribute `dsn`:<br />
Data source name, see driver document: [mysql](https://github.com/go-sql-driver/mysql/), [sqlite](https://github.com/mattn/go-sqlite3), [postgresql](https://github.com/lib/pq). e.g. (mysql) `root:password@tcp(127.0.0.1:3306)/test`

#### `database/query`

Sql to be executed. You can use `${param_name}` as a placeholder, and replace it by query parameters. 

### `user`

Defines a user and which resources who can access. Can appearances multiple times. 

#### `user/key`
Authorization key.

#### `user/host`
Host allowed access from by user. Can appearances multiple times.

#### `user/files`
Attribute `id`:<br/>
id of `files` can be access. Can appearances multiple times.

#### `user/commands`
Attribute `id`:<br/>
id of `commands` can be access. Can appearances multiple times.

#### `user/database`
Attribute `id`:<br/>
id of `database` can be access. Can appearances multiple times.

## client protocol

servant uses HTTP protocol. You can use `curl http://<host>:<port>/<resource_type>/<group>/<item>[/<sub item>]` to access resources., e.g. `curl http://127.0.0.1:2465/commands/db1/foo` to execute a command foo in db1 group.

### commands

only supports GET and POST method. 

#### simple
`curl http://127.0.0.1:2465/commands/db1/foo`

#### with input stream
`echo "hello world" | curl -XPOST http://127.0.0.1:2465/commands/db1/grep -d -`

#### with parameters
`curl http://127.0.0.1:2465/commands/db1/sleep?t=2`

### files

#### read a file
`curl http://127.0.0.1:2465/files/db1/binlog1/log-bin.000001`

#### read a file range
`curl -H 'Range: bytes=6-10' http://127.0.0.1:2465/files/db1/binlog1/test.txt`

#### create a file
`echo "hello world!" | curl -XPOST http://127.0.0.1:2465/files/db1/binlog1/test.txt -d -`

#### update a file
`echo "hello world!" | curl -XPUT http://127.0.0.1:2465/files/db1/binlog1/test.txt -d -`

#### delete a file
`curl -XDELETE http://127.0.0.1:2465/files/db1/binlog1/test.txt`

#### view file attributes
`curl -I http://127.0.0.1:2465/files/db1/binlog1/test.txt`

### databases
Output are in json format

`curl http://127.0.0.1:2465/databases/mysql/select_1`

`curl http://127.0.0.1:2465/databases/mysql/select_v?v=hello`


### authorization

servant uses a `Authorization` head to verify a user access. 

The format is `Authorization: <username> <timestamp> sha1(<username> + <key> + <timestamp> + <method> + <uri>)`.

Timestamp is a 32bit UNIX timestamp; method is in uppercase.

e.g.

    uri='commands/db1/foo'
    ts=$(date +%s)
    user=user1
    key=someKey
    curl -H "Authorization: user1 ${ts} $(echo "${user1}${key}${ts}GET${uri}"|sha1sum|cut -f1 -d' ')"  "http://127.0.0.1:2465/${uri}"
    
    