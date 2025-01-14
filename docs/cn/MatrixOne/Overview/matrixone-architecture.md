# **MatrixOne架构概述**

## **技术架构**

MatrixOne 是第一个开源的分布式 HSTAP 数据库，支持 OLTP、OLAP 和 Streaming，使用单个自动调优存储引擎。

HSTAP 对 HTAP 数据库进行了重新定义，HSTAP 旨在满足单一数据库内事务处理（TP）和分析处理（AP）的所有需求。与传统的 HTAP 相比，HSTAP 强调其内置的用于连接TP和AP表数据流处理能力。为用户提供了数据库可以像大数据平台一样的使用体验，也恰恰得益于大数据的繁荣，很多用户已经熟悉了这种体验。
MatrixOne 以最少的集成工作，让用户摆脱大数据的限制，为企业提供所有TP和AP场景的一站式覆盖。

MatrixOne 作为一个从零开始打造的全新数据库，并在其他 DBMS 中引入了多种创新和最佳实践。采用解耦存储和计算架构，将数据存储在 S3 云存储服务上，实现低成本，计算节点无状态，可随意启动，实现极致弹性。在本地部署可用的情况下，MatrixOne 可以利用用户的 HDFS 或其他支持的分布式文件系统来保存数据。MatrixOne 还配备了兼容 S3 的内置替代方案，以确保其存储的弹性、高可靠性和高性能，而无需依赖任何外部组件。架构如下：

![MatrixOne Architecture](https://github.com/matrixorigin/artwork/blob/main/docs/overview/matrixone_new_arch.png?raw=true)

## **集群管理层**

这一层负责集群管理，在云原生环境中与 Kubernetes 交互动态获取资源；在本地部署时，根据配置获取资源。集群状态持续监控，根据资源信息分配每个节点的任务。提供系统维护服务以确保所有系统组件在偶尔出现节点和网络故障的情况下正常运行，并在必要时重新平衡节点上的负载。集群管理层的主要组件是：

- Prophet 调度：提供负载均衡和节点 Keep-alive。
- 资源管理：提供物理资源。

## **Serverless 层**

Serverless 层是一系列无状态节点的总称，整体上包含三类：

- 后台任务：最主要的功能是 Offload Worker，负责卸载成本高的压缩任务，以及将数据刷新到S3存储。
- SQL计算节点：负责执行 SQL 请求，这里分为写节点和读节点，写节点还提供读取最新数据的能力。
- 流任务处理节点：负责执行流处理请求。

## **日志层**

作为 MatrixOne 的单一数据源 (即Single source of truth)，数据一旦写入日志层，则将永久地存储在 MatrixOne中。它建立在我们世界级的复制状态机模型的专业知识之上，以保证我们的数据具有最先进的高吞吐量、高可用性和强一致性。它本身遵循完全模块化和分解的设计，也帮助解耦存储和计算层的核心组件，与传统的 NewSQL 架构相比，我们的架构具有更高的弹性。

## **存储层**

存储层将来自日志层的传入数据转换为有效的形式，以供将来对数据进行处理和存储。包括为快速访问已写入 S3 的数据进行的缓存维护等。在 MatrixOne 中，TAE（即Transactional Analytic Engine）是存储层的主要公开接口，它可以同时支持行和列存储以及事务处理能力。此外，存储层还包括其他内部使用的存储功能，例如流媒体的中间存储。

## **存储供应层**

作为与基础架构解耦的 DBMS，MatrixOne 可以将数据存储在 S3/HDFS 、本地磁盘、本地服务器、混合云或其他各类型云，以及智能设备的共享存储中。存储供应层通过为上层提供一个统一的接口来访问这些多样化的存储资源，并且不向上层暴露存储的复杂性。

## **相关信息**

本节介绍了MatrixOne的整体架构概览。若您想了解更详细的模块技术设计问题，可阅读：
[MatrixOne模块概览](MatrixOne-Tech-Design/matrixone-techdesign.md)  

其他信息可参见：

* [安装MatrixOne](../Get-Started/install-standalone-matrixone.md)
* [MySQL兼容性](mysql-compatibility.md)
* [最新发布信息](whats-new.md)
