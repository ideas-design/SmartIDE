FROM ubuntu:20.04

# 所有操作都是非交互式的
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ Asia/Shanghai
#git中文乱码问题
ENV LESSCHARSET=utf-8


# sshd
RUN mkdir /var/run/sshd && \
    apt-get update && \
    apt-get -y install --no-install-recommends openssh-server lsof curl wget git sudo && \
    apt-get clean && \
    apt-get autoremove -y && \
    rm -rf /tmp/* /var/lib/apt/lists/* /var/tmp/*

EXPOSE 22

# install kubectl
RUN wget --no-check-certificate "https://smartidedl.blob.core.chinacloudapi.cn/kubectl/v1.23.0/bin/linux/amd64/kubectl" -O /usr/local/bin/kubectl 
RUN chmod +x /usr/local/bin/kubectl

# install htpasswd & ca-certificates
RUN apt-get update \
    && apt-get install apache2-utils -y \
    && apt-get install ca-certificates -y

COPY entrypoint_base.sh /idesh/entrypoint_base.sh
RUN chmod +x /idesh/entrypoint_base.sh

ENTRYPOINT ["/idesh/entrypoint_base.sh"]