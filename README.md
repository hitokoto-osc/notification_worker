# Notification Worker
## 简介
如其名，该项目最初目的是作为发送通知的队列消费者。抽象后，可以作为广泛用途的 RabbitMQ 消费服务（基于事件模型）

### 项目运用的第三方库
* RabbitMQ 官方提供的 amqp 库
* Aliyun DirectMail SDK

## 使用
首先克隆本项目到你的工作目录下，然后编辑代码，即可。这里推荐使用 GoLand 编码。

## 编译

本项目提供两种编译方法：

1. 在项目根目录使用 `make build` 即可自动编译。（支持 linux macos）

2. 只需要执行 ./build.sh 即可自动编译。（需要 Linux 环境）