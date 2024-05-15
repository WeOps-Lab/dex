## 嘉为蓝鲸docker插件使用说明

## 使用说明

### 插件功能

Docker Exporter是一个用于监控docker容器的工具，依赖docker.sock获取监控数据。  

### 版本支持

操作系统支持: linux, windows

是否支持arm: 支持

**组件支持版本：**

docker版本:  
API version: 1.24 ~ 1.45  
Docker version: 1.12 ~ 26.1

**是否支持远程采集:**

否

### 参数说明


| **参数名**              | **含义**            | **是否必填** | **使用举例**       |
|----------------------|-------------------|----------|----------------|
| --debug              | 是否进行debug         | 否        | --debug=true   |
| --web.listen-address | exporter监听id及端口地址 | 否        | 127.0.0.1:9601 |

### 使用指引
直接下发探针进行采集  

1. 查看docker版本信息
```shell
### 输入查看版本信息的命令
docker version

### 返回的版本信息
Client:
 Version:           20.10.8
 API version:       1.41
 Go version:        go1.16.6
 Git commit:        3967b7d
 Built:             Fri Jul 30 19:50:40 2021
 OS/Arch:           linux/amd64
 Context:           default
 Experimental:      true

Server: Docker Engine - Community
 Engine:
  Version:          20.10.8
  API version:      1.41 (minimum version 1.12)
  Go version:       go1.16.6
  Git commit:       75249d8
  Built:            Fri Jul 30 19:55:09 2021
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          v1.4.9
  GitCommit:        e25210fe30a0a703442421b0f60afac609f950a3
 runc:
  Version:          1.0.1
  GitCommit:        v1.0.1-0-g4144b638
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
```


### 指标简介

| **指标ID**                     | **指标中文名** | **维度ID**                    | **维度含义**   | **单位**  |
|------------------------------|-----------|-----------------------------|------------|---------|
| up                           | 监控探针运行状态  | -                           | -          | -       |
| container_running            | 容器运行状态    | container_name, image       | 容器名称, 镜像名称 | -       |
| cpu_utilization_percent      | CPU使用率    | container_name, image       | 容器名称, 镜像名称 | percent |
| memory_total_bytes           | 限制内存使用大小  | container_name, image       | 容器名称, 镜像名称 | bytes   |
| memory_usage_bytes           | 使用内存大小    | container_name, image       | 容器名称, 镜像名称 | bytes   |
| memory_utilization_percent   | 内存使用率     | container_name, image       | 容器名称, 镜像名称 | percent |
| container_volume_usage_bytes | 容器存储使用字节数 | container_name, volume_name | 容器名称, 卷名称  | bytes   |
| volume_usage_bytes           | 存储使用字节数   | volume_name                 | 卷名称        | bytes   |



### 版本日志

#### weops_docker_exporter 2.3.5

- weops调整


添加“小嘉”微信即可获取docker监控指标最佳实践礼包，其他更多问题欢迎咨询

<img src="https://wedoc.canway.net/imgs/img/小嘉.jpg" width="50%" height="50%">
