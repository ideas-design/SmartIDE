---
title: "SmartIDE v0.1.5 发布说明"
linkTitle: "版本 0.1.5 发布说明"
date: 2021-11-05
description: >
  SmartIDE v0.1.3 发布说明。远程主机模式完善，支持Git认证。
---

首先说明一下版本号的变更，你也需注意到了我们的版本号从v0.1.2直接跳到了v0.1.5，这是因为我们决定使用第三位版本号代表我们的迭代号，所以v0.1.5就代表这是sprint 5的交付版本。我们在sprint 4中完善了SmartIDE的分支策略和流水线工作模式，为了能够更加明确的跟踪我们的版本，因此在Sprint 5中我们决定采用第三位版本号来跟踪迭代号码，后续我也会专门提供一篇文档/博客来说明SmartIDE团队是如何管理自己的backlog, kanban，开发环境以及流水线的。

在刚刚结束的Sprint 5中，我们着重对SmartIDE的远程主机模式进行了一系列的改进，让远程主机模式进入可用状态，以下是Sprint 5所交付特性的列表：

- 完善远程主机连接方式：允许用户选择用户名密码或者SSH密钥两种方式连接到远程主机
- SSH密钥同步：允许用户将本地的SSH密钥同步到远程主机上，省去用户重新设置SSH密钥的步骤
- Git URL改进：允许用户通过https/SSH两种GitURL连接到自己的Git仓库，同时完善了私有Git库的支持

```bash
# 打开终端窗口，复制粘贴以下脚本即可安装稳定版SmartIDE CLI应用
curl -OL  "https://smartidedl.blob.core.chinacloudapi.cn/releases/$(curl -L -s https://smartidedl.blob.core.chinacloudapi.cn/releases/stable.txt)/smartide" \
&& mv -f smartide /usr/local/bin/smartide \
&& chmod +x /usr/local/bin/smartide
```