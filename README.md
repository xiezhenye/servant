# servant
A common agent to execute configured command and serve file read write via HTTP protocol

## build
### executable
    cd <path to project>
    make
### rpm
    cd <path to project>
    make rpm
### tarball
    cd <path to project>
    make tarball
    
By defaults, only mysql database driver are built in. You can use `make DRIVERS="mysql sqlite postgres"` to choose other drivers.

## usage
    /path/to/servant/scripts/servantctl (start|stop|restart|status|help)

## command-line arguments

    Usage of ./servant:
        -conf value
                config files path
        -confdir value
                config directories path
        -var value
                vars

 * -conf

    Config file path. can presents multiple times.

 * -confdir

    Config directories path. All config files in it will be loaded. Can presents multiple times.

 * -var

    Predefined vars. e.g. `-var foo=bar` can be referenced as `${_arg.foo}`. Can presents multiple times.


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
            <log>servant.log</log>
        </server>

        <commands id="db1">
            <command id="foo" runas="mysql" lang="bash">
                <code>echo "hello world $(whoami)"</code>
            </command>
            <command id="grep" lang="exec">
                <code>grep hello</code>
            </command>
            <command id="sleep" timeout="5" lang="exec">
                <code> sleep ${t}</code>
            </command>
        </commands>

        <daemon id="daemon1" retries="10" lang="bash">
            <code>sleep 10000</code>
        </daemon>

        <timer id="xx" tick="5" deadline="5" lang="bash">
            <code>
            <![CDATA[
                 date >>/tmp/timer.log
            ]]>
            </code>
        </timer>

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
            <query id="select_1"><sql> select 1 </sql></query>
        </database>

        <vars id="vars">
            <var id="foo">
                <value>bar</value>
            </var>
            <var id="hello" expand="true">
                <value>${world}</value>
            </var>
        </vars>

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

* Attribute `enabled`:

  can be 0 or 1. When authorization disabled, `user` config has no use.

* Element `server/auth/maxTimeDelta`:

  Max time delta between servant server and client allowed.

#### `server/log`

Log file path. If not set, log will be writen to stdout.

### resources group elements

Resources group elements can be `commands`, `files`, `database`, `vars` which defines some resource item elements. Each resource group and resource item elements must has an `id` attribute. Client can reference a resource by `/<resource_type>/<group>/<item>`, e.g. `/commands/db1/foo`. `daemon`, `timer` does not has a group, they are defined directly under `server` element.

### `commands`

Defines a group of commands can be executed, contains some `command` elements.

#### `commands/command`
* Attribute `lang`: 

  Can be `bash`, `exec`. <br />
  As `bash`, code is executed as bash, supports if, for, pipe etc, but not supports parameter replacement. <br />
  As `exec`, code implies just a single command and arguments separated by spaces. You can use `${param_name}` as a placeholder, and replace it by query parameters. If you want to use an argument contains spaces, you can wrap it by quotes e.g. "hello world", 'hello servant'.

* Attribute `runas`:

  The system user to execute the command, default is current user. To use `runas`, servant must be run as root.

* Attribute `timeout`:
  
  Limit the command execution time in seconds, default is unlimited.

* Attribute `background`:

  Whether the command runs in background. Could be true or false. When `background` == true, Servant will return immediately.

* Element `code`:

  Code of the command to be executed

* Element `lock`:

  Mutex. Attributes: name: locks with same name is exclusive. wait: when race for the lock failed, wait until the lock is released or return immediately, default is false. timeout: Max time to wait for the lock, in seconds. 

* Element `validate`:

  Validate params. Attributes: name: param name to validate. Body: Validator regexp.  


### `daemon`
* Attribute `lang`:

  Can be `bash`, `exec`. see `commands/command`

* Attribute `runas`:

  The system user to execute the command. see `commands/command`

* Attribute `retries`:

  Retry times when run code failed.

* Attribute `live`:

  Seconds a daemon runs before failed to reset retry counter. Default is unlimited. 

* Element `code`:

  Code of the command to be executed

### `timer`
* Attribute `lang`:

  Can be `bash`, `exec`. see `commands/command`

* Attribute `runas`:

  The system user to execute the command. see `commands/command`

* Attribute `tick`:

  Interval in seconds to trigger the timer

* Attribute `deadline`:

  Seconds of the max duration the timer task can runs.

* Element `code`:

  Code of the command to be executed

### `files`

Defines some directories can be accessed.
 
#### `files/dir`

A directory can be accessed.

* Element `root`:

  The root of the directory. Access will be limited in it.

* Element `allow`:

  Methods allows to access the files in the directory. can be get, post, put, delete, head. Which means read, create, update, delete, stat. This element can appearances more than one times. 

* Element `pattern`:

  File name patterns allowed to be access in regular expression. If not defined, all files in the directory can be accessed. This element can appearances more than one times. 

* Element `validate`:

  Validate params. Attributes: name: param name to validate. Body: Validator regexp.  


### `database`

A database resource.

* Attribute `driver`:

  Database driver, supports mysql, sqlite, postgresql. 

* Attribute `dsn`:

  Data source name, see driver document: [mysql](https://github.com/go-sql-driver/mysql/), [sqlite](https://github.com/mattn/go-sqlite3), [postgresql](https://github.com/lib/pq). e.g. (mysql) `root:password@tcp(127.0.0.1:3306)/test`

#### `database/query`

Sqls to be executed. Will be executed during a database session.

* Element `sql`:

  A sql. You can use `${param_name}` as a placeholder, and replace it by query parameters.  Can appearances multiple times.

* Element `validate`:

  Validate params. Attributes: name: param name to validate. Body: Validator regexp.  

### `vars`

Defines a group of variables. 

Variables expand can be used in `command`, `var`, `file/root`. `${param_name}` is a request param, `${group.item}` is a user define varaible, `${_arg.name}` is a command-line argument variable. Variable expand can also defined recursively, like `${group.${item_param}}`


#### `vars/var`

A variable

* Attribute `expand`:

  Whether expands variables in value or not.

* Attribute `readonly`:

  Whether value can be updated online or not.

* Element `value`:

  Default variable value.

* Element `pattern`:

  Regexp pattern the value must matchs.

### `user`

Defines a user and which resources who can access. Can appearances multiple times. 

#### `user/key`
Authorization key.

#### `user/host`
Host allowed access from by user. Can appearances multiple times.

#### `user/files`
* Attribute `id`:

  id of `files` can be access. Can appearances multiple times.

#### `user/commands`
* Attribute `id`:

  id of `commands` can be access. Can appearances multiple times.

#### `user/databases`
* Attribute `id`:

  id of `database` can be access. Can appearances multiple times.

## client protocol

servant uses HTTP protocol. You can use `curl http://<host>:<port>/<resource_type>/<group>/<item>[/<sub item>]` to access resources., e.g. `curl http://127.0.0.1:2465/commands/db1/foo` to execute a command foo in db1 group.

### commands

only supports GET and POST method. 

#### simple
`curl http://127.0.0.1:2465/commands/db1/foo`

#### with input stream
`echo "hello world" | curl -XPOST http://127.0.0.1:2465/commands/db1/grep -d @-`

#### with parameters
`curl http://127.0.0.1:2465/commands/db1/sleep?t=2`

### files

#### read a file
`curl http://127.0.0.1:2465/files/db1/binlog1/log-bin.000001`

#### read a file range
`curl -H 'Range: bytes=6-10' http://127.0.0.1:2465/files/db1/binlog1/test.txt`

#### create a file
`echo "hello world!" | curl -XPOST http://127.0.0.1:2465/files/db1/binlog1/test.txt -d @-`

#### update a file
`echo "hello world!" | curl -XPUT http://127.0.0.1:2465/files/db1/binlog1/test.txt -d @-`

#### delete a file
`curl -XDELETE http://127.0.0.1:2465/files/db1/binlog1/test.txt`

#### view file attributes
`curl -I http://127.0.0.1:2465/files/db1/binlog1/test.txt`

### databases
Outputs are in json format

`curl http://127.0.0.1:2465/databases/mysql/select_1`

`curl http://127.0.0.1:2465/databases/mysql/select_v?v=hello`

### variables

#### get a variable

`curl http://127.0.0.1:2465/vars/foo`

#### set a variable

`curl  -XPOST http://127.0.0.1:2465/vars/foo -d 'BAR'`

### authorization

servant uses a `Authorization` head to verify a user access. 

The format is `Authorization: <username> <timestamp> sha1(<username> + <key> + <timestamp> + <method> + <uri>)`.

Timestamp is a 32bit UNIX timestamp; method is in uppercase.

e.g.

    uri='/commands/db1/foo'
    ts=$(date +%s)
    user=user1
    key=someKey
    curl -H "Authorization: ${user} ${ts} $(echo -n "${user}${key}${ts}GET${uri}"|sha1sum|cut -f1 -d' ')"  "http://127.0.0.1:2465${uri}"
    
    
