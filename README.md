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

2.3）还原docker镜像
1、安装rar解压软件
a) sudo apt-get install rar
b) ln -fs /usr/bin/rar /usr/bin/unrar

2、从github上克隆文件(如未安装git,可执行sudo apt-get install git进行git安装)
进入ubutu根目录，假设ubuntu用户名为rblockchain
a)  cd /home/rblockchain下执行
b)  git clone https://github.com/rblockchain/eloan-peer.git

3、解压从github上克隆的镜像
a)  在cd /home/rblockchain下执行
b)  rar x FileName.rar

注意：从fabric-baseimage.part01.tar至fabric-baseimage.part07.tar，仅需任意解压一个rar文件即可。例如：rar x fabric-baseimage.part01.tar。同理，从fabric-javaenv.part01.tar至fabric-baseimage.part05.tar也仅需解压一个即可

4、打包解压出的文件
rar文件解压完毕后，需要进入到解压生成的文件夹内部执行命令：
a)  在cd /home/rblockchai/fabric-javaenv下执行
b)  tar cvf fabric-javaenv.tar *
同理，fabric-baseimage、fabric-baseos、fabric-peer相对应进入到fabric-baseimage、fabric-baseos、fabric-peer下执行tar命令生成相应的.tar包

5、还原docker镜像
a)  cd /home/rblockchai/fabric-javaenv
b)  docker load<fabric-javaenv.tar
同理，fabric-baseimage、fabric-baseos、fabric-peer也需执行docker load命令还原出docker镜像
6、检查镜像是否导出成功
执行docker images命令，确认镜像是否导出成功

2.4）docker compose部署
参考文档：http://blog.csdn.net/yl_1314/article/details/53761049

1、root权限下执行：curl -L "https://github.com/docker/compose/releases/download/1.9.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

2、在/usr/local/bin目录下，执行：chmod +x docker-compose

3、检查docker compose 是否部署正常
docker-compose version