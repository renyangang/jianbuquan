# jianbuquan

代码结构介绍：
1. 目录结构 -- page[存放html页面模板] pkg[存放自身和第三方库静态库，编译时生成] public[存放web静态内容] src[存放源代码]
2. 代码结构 -- main.go为入口文件。 dataobj包定义数据模型 webhandler包处理页面逻辑 weblog包为日志模块 weixin包实现了和微信接口协议对接 另外还应用了第三方开源包redigo用于和redis通信
3. 运行结构 -- go语言实现全部页面和微信对接逻辑，数据使用redis存储，数据模型定义参见redis设计.xlsx
               交互流程如下：
               公众号<---->web服务器<---->redis
               
代码编译：
1. 环境准备： 可运行与linux或者windows系统，需要安装go语言环境，git工具
2. clone项目 git clone https://github.com/renyangang/jianbuquan.git
3. 下载redigo项目，可以使用go get下载，也可以直接下载zip包，解压到src目录下。
4. 设置GOPATH环境变量为 src上一级目录， 在src下执行go build 编译

服务运行：
1. 安装redis服务器，要求监听 127.0.0.1:8432
2. 编译好的服务器程序在 public page 平级的目录运行即可。 生成web.log日志文件。