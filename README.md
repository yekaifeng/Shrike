# D4D (DHCP for Docker)



### 开发背景

众所周知，Docker容器跨主机互访一直是一个问题，Docker官方为了避免网络上带来的诸多麻烦，故将跨主机网络开了比较大的口子，而由用户自己去实现。

目前Docker跨主机的网络实现方案也有很多种, 主要包括端口映射，ovs, fannel等。但是这些方案都无法满足我们的需求，端口映射服务内的内网IP会映射成外网的IP，这样会给开发带来困惑，因为他们往往在跨网络交互时是不需要内网IP的，而ovs与fannel则是在基础网络协议上又包装了一层自定义协议，这样当网络流量大时，却又无端的增加了网络负载，最后我们采取了自主研发扁平化网络插件，也就是说让所有的容器统统在大二层上互通。

感谢talkingdata, 本项目源自Shrike, 并对IP分配机制作出改动, 改为使用DHCP服务为容器分配IP.

### 安装部署范例


假设我们有2台物理机想要部署docker

DOCKER: 192.168.59.202

DHCP SERVER: 192.168.59.203


##### DHCP服务

首先，我们需要安装DHCP服务,提供IP地址分配功能

我们使用下面命令分别在这3台机器上运行

```shell
yum install -y dhcp
```

安装完成后，修改/etc/dhcp/dhcpd.conf 文件
这个例子使用192.168.59.10 - 192.168.59.30 作为容器网段

```

    #
    # DHCP Server Configuration file.
    #   see /usr/share/doc/dhcp*/dhcpd.conf.example
    #   see dhcpd.conf(5) man page
    #
    # dhcpd.conf
    #
    # Sample configuration file for ISC dhcpd
    #
    
    # option definitions common to all supported networks...
    option domain-name "example.org";
    option domain-name-servers ns1.example.org, ns2.example.org;
    
    default-lease-time 600;
    max-lease-time 7200;
    
    # Use this to enble / disable dynamic dns updates globally.
    #ddns-update-style none;
    
    # If this DHCP server is the official DHCP server for the local
    # network, the authoritative directive should be uncommented.
    #authoritative;
    
    # Use this to send dhcp log messages to a different log file (you also
    # have to hack syslog.conf to complete the redirection).
    log-facility local7;
    
    # No service will be given on this subnet, but declaring it helps the 
    # DHCP server to understand the network topology.
    
    #subnet 10.152.187.0 netmask 255.255.255.0 {
    #}
    
    # This is a very basic subnet declaration.
    
    #subnet 10.254.239.0 netmask 255.255.255.224 {
    #  range 10.254.239.10 10.254.239.20;
    #  option routers rtr-239-0-1.example.org, rtr-239-0-2.example.org;
    #}
    
    # This declaration allows BOOTP clients to get dynamic addresses,
    # which we don't really recommend.
    
    #subnet 10.254.239.32 netmask 255.255.255.224 {
    #  range dynamic-bootp 10.254.239.40 10.254.239.60;
    #  option broadcast-address 10.254.239.31;
    #  option routers rtr-239-32-1.example.org;
    #}
    
    # A slightly different configuration for an internal subnet.
    subnet 192.168.59.0 netmask 255.255.255.0 {
      range 192.168.59.10 192.168.59.30;
      option domain-name-servers ns1.internal.example.org;
      option domain-name "internal.example.org";
      option routers 192.168.59.1;
      option broadcast-address 192.168.59.255;
      default-lease-time 600;
      max-lease-time 7200;
    }
    
```

修改完配置文件后，运行命令

```shell
systemctl start dhcpd
```


##### Docker

在第一台机器上安装docker,需使用docker v1.12以上

```shell
yum localinstall -y ./rpms/docker-engine-1.12.1-1.el7.centos.x86_64.rpm
yum localinstall -y ./rpms/docker-engine-selinux-1.12.1-1.el7.centos.x86_64.rpm
```

安装完成后，修改/usr/lib/systemd/system/docker.service 文件

```
[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network.target docker.socket
Requires=docker.socket

[Service]
Type=notify
# the default is not to use systemd for cgroups because the delegate issues still
# exists and systemd currently does not support the cgroup feature set required
# for containers run by docker
ExecStart=/usr/bin/docker daemon --debug -H=fd:// -H=tcp://0.0.0.0:2375 -H=unix:///var/run/docker.sock --insecure-registry=172.31.0.110:5000 -s=overlay -g /ssdcache/docker
MountFlags=slave
LimitNOFILE=1048576
LimitNPROC=1048576
LimitCORE=infinity
TimeoutStartSec=0
# set delegate yes so that systemd does not reset the cgroups of docker containers
Delegate=yes

[Install]
WantedBy=multi-user.target
```

参数--insecure-registry是镜像管理地址，这个要根据情况进行更改，以上配置都是符合我们需求的。

修改完配置后，运行

```shell
systemctl start docker
```



##### OAM-DOCKER-IPAM

我们需要在192.168.59.202机器上安装oam-docker-ipam插件

```shell
yum localinstall -y ./rpms/oam-docker-ipam-2.0.0-1.el7.centos.x86_64.rpm
```

安装完成后，修改/etc/oam-docker-ipam/oam-docker-ipam.conf 文件

```
# [ipam]
# [ipam]
IPAM_DEBUG=true
DHCP_SERVER=192.168.59.203  #DHCP服务器地址
LISTEN_ADDR=0.0.0.0         #本机用于请求DHCP的地址,缺省为所有端口地址
```

修改完配置后，运行

```shell
systemctl start oam-docker-ipam
# systemctl status oam-docker-ipam
● oam-docker-ipam.service - oam-docker-ipam
   Loaded: loaded (/usr/lib/systemd/system/oam-docker-ipam.service; disabled; vendor preset: disabled)
   Active: active (running) since 二 2017-01-17 03:52:24 EST; 38min ago
 Main PID: 3031 (oam-docker-ipam)
   Memory: 3.1M
   CGroup: /system.slice/oam-docker-ipam.service
           └─3031 /usr/bin/oam-docker-ipam --debug=true --dhcp-server=192.168.59.203 --listen-addr=0.0.0.0 server
```

##### 创建网络并运行容器

```shell
docker network create 
-d macvlan 
--subnet=192.168.59.0/24 
--gateway=192.168.59.1 
--ipam-driver=talkingdata 
--aux-address="DefaultGatewayIPv4=192.168.59.1" 
-o parent=enp0s8 macvlan
docker run -d --name 1 --net macvlan --privileged centos:latest /bin/bash -c 'while true;do echo test;sleep 90;done'
```

### 如何构建RPM

```shell
./build-rpm.sh oam-docker-ipam 2.0.1
```

构建需要Go1.5环境, rpmbuild命令,生成rpm放在rpms目录

### 有关功能限制

- 因为d4d需要绑定68端口作为dhcp客户端发请求, 宿主机需要关闭dhclient功能,否则会互相冲突
