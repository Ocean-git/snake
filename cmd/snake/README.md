# snake 

## 脚手架介绍

### snake 脚手架工具集

1. 快速生成模板项目
2. ...

## Go 版本要求

GO >= 1.13

## 脚手架工具获取

windows :

```bash
set GO111MODULE=on
set GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

linux :

```bash
export GO111MODULE=on
export GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

下载

```bash
go get -u -v github.com/1024casts/snake/cmd/snake
```

windows:
 会在 `${GOPATH}/src/bin` 目录下生成 `snake.exe` 文件,若想方便的在任何地方使用 `snake` 命令,请将该 命令配置在系统的环境变量中
Linux、Mac:
可以直接使用 `snake`

## 使用方式

- snake -h

```bash
$ snake -h
NAME:
   snake - snake tools

USAGE:
   cmd [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
     new, n   Create Snake template project
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

- snake new -h 

```bash
$ snake new -h
NAME:
   cmd new - Create Snake template project

USAGE:
   
snake [commands|flags]
The commands & flags are:
  new     Create Snake template project
  -d      Build the specified directory for the template project
Examples:
  # Build the specified directory for the template project
  snake new (your project name) -d (project dir)


OPTIONS:
   -d value  Specify the directory of the project
```

## 快速创建 snake 模板项目

```bash
cd ${GOPATH}/src
snake new snake-demo -d ./
```

命令解释:

- new :创建 snake 模板项目
- snake-demo: 项目名称
- -d: 生成项目所在路径

然后就会在 `${GOPATH}/src` 下生成 `snake-demo` 项目
项目目录结构:

```bash
├── Makefile                     # 项目管理文件
├── build                        # 编译目录
├── cmd                          # 脚手架目录
├── conf                         # 配置文件统一存放目录
├── config                       # 专门用来处理配置和配置文件的 Go package
├── db.sql                       # 在部署新环境时，可以登录 MySQL 客户端，执行 source db.sql 创建数据库和表
├── docs                         # Swagger 文档，执行 swag init 生成的
├── handler                      # 控制器目录，用来读取输入、调用业务处理、返回结果
├── internal                     # 业务目录
│   ├── cache                    # 基于业务封装的cache
│   ├── idl                      # 数据结构转换
│   ├── model                    # 数据库 model
│   ├── repository               # 数据访问层
│   └── service                  # 业务逻辑层
├── logs                         # 存放日志的目录
├── main.go                      # 项目入口文件
├── router                       # 路由及中间件目录
└── scripts                      # 存放常用脚本
```