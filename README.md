# eloan-peer
一、环境部署所需软件清单
•	Docker - v1.12 or higher
•	Docker Compose - v1.8 or higher
•	Go - 1.7 or higher
•	Git Bash - Windows users only; provides a better alternative to the Windows command prompt
本文档主要提供在ubuntu 14.04下的环境部署具体步骤。
二、安装具体步骤
2.1） go环境
通过ssh上传go1.8.linux-amd64.tar.gz到/opt文件夹下
tar zxvf go1.8.linux-amd64.tar.gz 

vi /etc/profile
添加： 
export GOROOT=/opt/go
export GOPATH=/opt/gopath
export GOBIN=$GOROOT/bin
export GOOS=linux
export GOARCH=amd64
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

加载
source /etc/profile

解决vim方向键变成ABCD
1. echo "set nocp" >> ~/.vimrc    （千万要注意，是>>, 而不是>, 否则把.vimrc清空了， 丢失了之前的内容）
2. source ~/.vimrc  


2.2）docker环境
参考：http://blog.csdn.net/zhangchao19890805/article/details/52849404

使用命令 uname -r 确保比版本比3.10高。 
更新安装包 sudo apt-get update 

sudo apt-get install apt-transport-https ca-certificates
sudo apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D

创建 /etc/apt/sources.list.d/Docker.list 文件
内容：deb https://apt.dockerproject.org/repo ubuntu-trusty main

安装 Docker 依赖的库和 Docker 程序本身：
sudo apt-get update && sudo apt-get purge lxc-docker
sudo apt-get update && sudo apt-get install linux-image-extra-$(uname -r) linux-image-extra-virtual
sudo apt-get update && sudo apt-get install docker-engine

echo "DOCKER_OPTS=\"\$DOCKER_OPTS --registry-mirror=https://c1pdyrea.mirror.aliyuncs.com\"" | sudo tee -a /etc/default/docker
检查：vi /etc/default/docker
DOCKER_OPTS="$DOCKER_OPTS --registry-mirror=https://c1pdyrea.mirror.aliyuncs.com"

service docker restart

免 sudo 使用 docker

如果还没有 docker group 就添加一个
sudo groupadd docker

ubuntu下，通过一下命令来看有没有group
cat /ect/group

将用户加入该 group 内。然后退出并重新登录就生效啦
sudo gpasswd -a ${USER} docker

重启 docker 服务
sudo service docker restart

group 或者重启 X 会话
newgrp - docker 或者 pkill X

docker compose部署
参考文档：http://blog.csdn.net/yl_1314/article/details/53761049

1、root权限下执行：curl -L "https://github.com/docker/compose/releases/download/1.9.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

2、在/usr/local/bin目录下，执行：chmod +x docker-compose

3、检查docker compose 是否部署正常
docker-compose version
