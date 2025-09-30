# Clash Subscription Converter

仅支持 Clash 的订阅转换器。订阅请求、Header 透传、规则集下载缓存由 Go 实现，具体订阅转换逻辑由 JS 脚本提供。具体转换逻辑的开发（JS
脚本）不是本仓库的核心目标，故仅提供示例。

## 功能特性

- **配置灵活**：转换逻辑由 JS 脚本定义
- **多订阅合并**：支持合并多个订阅源的节点
- **流量统计**：自动解析和合并订阅流量信息
- **规则缓存**：规则集和模板文件自动缓存，减少网络请求
- **Web UI**：提供友好的前端界面，快速生成订阅链接

## 快速开始

### 环境要求

- Go 1.16+

### 编译运行

```bash
# 克隆仓库
git clone https://github.com/yourusername/clash-converter.git
cd clash-converter

# 编译
go build

# 运行
./clash-converter
```

服务将在 `http://localhost:8080` 启动。

### Docker 部署

**使用预构建镜像：**

```bash
# 拉取镜像
docker pull ghcr.io/etnatker/clash-converter:latest

# 运行容器
docker run -d \
  --name clash-converter \
  -p 8080:8080 \
  -e ACCESS_TOKEN=your-secret-token \
  -v $(pwd)/data:/app/data \
  ghcr.io/etnatker/clash-converter:latest
```

**使用 Dockerfile 构建：**

```bash
# 构建镜像
docker build -t clash-converter .

# 运行容器
docker run -d \
  --name clash-converter \
  -p 8080:8080 \
  -e ACCESS_TOKEN=your-secret-token \
  -v $(pwd)/data:/app/data \
  clash-converter
```

**使用 docker-compose：**

```yaml
version: '3'
services:
  clash-converter:
    build: .
    # 或使用预构建
    # image: ghcr.io/etnatker/clash-converter:latest
    ports:
      - "8080:8080"
    environment:
      - ACCESS_TOKEN=your-secret-token
      - CACHE_EXPIRE_SEC=86400
    volumes:
      - ./data:/app/data
    restart: unless-stopped
```

启动：`docker-compose up -d`

### 环境变量

```bash
# 缓存过期时间（秒），默认 86400（24小时）
export CACHE_EXPIRE_SEC=86400

# 数据库路径，默认 ./data/database.db
export DB_PATH=./data/database.db

# 访问令牌（强烈建议设置）
export ACCESS_TOKEN=your-secret-token
```

## API 文档

### GET /sub

订阅转换接口，支持多订阅合并。

**请求参数：**

| 参数       | 类型       | 必填 | 说明          |
|----------|----------|----|-------------|
| sub      | string[] | 是  | 订阅链接，可传入多个  |
| script   | string   | 是  | JS 脚本 URL   |
| template | string   | 是  | 模板 YAML URL |
| token    | string   | 是  | 访问令牌        |


**响应：**

- 成功：返回转换后的 Clash 配置（YAML 格式）
- 失败：返回错误信息

**响应头：**

- `Content-Disposition`: 合并后的订阅文件名（用`|`分隔的各订阅名）
- `Subscription-Userinfo`: 合并后的流量统计信息

### GET /ui

Web 界面，用于可视化生成订阅链接。

访问 `http://localhost:8080/ui` 即可使用。

**URL 参数预填：**

所有配置项都可以通过 URL 参数传入并预填写（用于保存至书签或分享）。

### GET /ping

健康检查接口。

**响应：** `pong`

## 前端 UI 使用

1. 访问 `http://localhost:8080/ui`
2. 填写以下配置：
    - **Base URL**：订阅服务的基础地址（默认为当前页面域名）
    - **订阅列表**：添加一个或多个订阅链接，支持排序
    - **Script URL**：JS 脚本地址
    - **Template URL**：模板文件地址
    - **Access Token**：访问令牌
3. 页面会生成两个链接：
    - **订阅链接**：用于 Clash 客户端订阅
    - **收藏链接**：包含当前配置的页面链接，可保存到书签

## 订阅用量信息

转换后的配置会在最前面自动添加一个 **"Sub Info"** 节点组，用于显示各订阅的用量信息。

**节点命名格式：**

```
订阅名称：已用GB/总量GB
```

例如：`订阅01：12.5/100.0`，其中`订阅01`为缺省名称。订阅若下发文件名，则会采用下发的文件名。

## Template 模板文件

Template 是一个 YAML 格式的 Clash 配置文件，定义基础配置和策略组结构。程序会自动填充以下内容：

- **proxies**：订阅的所有节点（包括 Sub Info 假节点）
- **rules**：根据 JS 脚本定义的规则集生成的规则列表

模板中可以定义：

- 基础配置（port、mode、log-level 等）
- 策略组（proxy-groups）
- DNS 配置
- 其他 Clash 支持的配置项

**说明**：

- `proxies` 和 `rules` 字段会被覆盖
- JS 脚本的 `buildConfig()` 可以进一步修改模板生成的配置

## JS 脚本

JS 脚本负责定义订阅转换的具体逻辑，包括规则集的下载和配置的最终调整。脚本在 Go
运行时中通过 [goja](https://github.com/dop251/goja) 引擎执行。

### 执行流程

1. Go 从订阅 URL 获取节点列表
2. 执行 JS 脚本，调用 `rulesets()` 函数获取规则集列表
3. Go 并发下载所有规则集（支持缓存）
4. Go 根据模板、节点和规则集构建基础配置
5. 执行 JS 的 `buildConfig()` 函数（如果存在）进行最终调整
6. 返回最终配置

### 需要实现的函数

#### rulesets(callback)

用于定义需要下载的规则集。

- **参数**：`callback(tag, url)` - 规则集回调函数
    - `tag` (string): 规则标签，将作为规则的目标策略组
    - `url` (string): 规则集文件的 URL（支持缓存）
- **返回值**：无

**示例：**

```javascript
function rulesets(callback) {
    // 定义直连规则
    callback('DIRECT', 'https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list');
    callback('DIRECT', 'https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list');

    // 定义代理规则
    callback('PROXY', 'https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyLite.list');
    callback('PROXY', 'https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Telegram.list');

    // 定义特定应用规则
    callback('Netflix', 'https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Netflix.list');
}
```

**规则处理**：

- 规则集文件中的规则会被解析并添加 `tag` 作为目标
- 例如：`DOMAIN,google.com` → `DOMAIN,google.com,PROXY`
- 支持的规则格式参考 Clash Meta 文档

#### buildConfig(config)

用于在配置生成后进行最终调整。

- **参数**：`config` (object) - 完整的 Clash 配置对象（可修改）
- **返回值**：无（直接修改 `config` 对象）

**config 结构示例：**

```javascript
{
    "proxies": [...],           // 节点列表（包含 Sub Info 假节点）
    "proxy-groups": [...],      // 策略组（包含 Sub Info 组）
    "rules": [...],             // 规则列表
    "mixed-port": 7890,         // 其他配置项...
    // ... 模板中的其他配置
}
```

**示例：**

```javascript
function buildConfig(config) {
    // 修改端口
    config['mixed-port'] = 7890;
    config['allow-lan'] = true;

    // 添加自定义 DNS 配置
    config['dns'] = {
        enable: true,
        listen: '0.0.0.0:53',
        nameserver: ['223.5.5.5', '119.29.29.29']
    };

    // 添加自定义策略组
    config['proxy-groups'].push({
        name: 'Custom',
        type: 'select',
        proxies: ['DIRECT', 'PROXY']
    });

    // 在规则列表最后添加自定义规则
    config['rules'].push('MATCH,PROXY');
}
```

### 可用的内置函数

#### log(message)

输出日志到服务端控制台，用于调试。

- **参数**：`message` - 要输出的内容
- **返回值**：无

**示例：**

```javascript
log('开始处理规则集');
log('规则数量: ' + config.rules.length);
log({status: 'ok', count: 42});  // 支持对象
```

**日志输出**：日志会以 `[JS]` 前缀显示在服务端控制台。

### 完整示例

```javascript
// 定义规则集
function rulesets(callback) {
    log('开始加载规则集');

    // 基础规则
    callback('DIRECT', 'https://example.com/direct.list');
    callback('PROXY', 'https://example.com/proxy.list');
    callback('REJECT', 'https://example.com/reject.list');

    // 流媒体规则
    callback('Netflix', 'https://example.com/netflix.list');
    callback('Disney+', 'https://example.com/disney.list');

    log('规则集配置完成');
}

// 自定义配置
function buildConfig(config) {
    log('开始自定义配置');

    // 基础设置
    config['mixed-port'] = 7890;
    config['allow-lan'] = true;
    config['log-level'] = 'info';

    // DNS 配置
    config['dns'] = {
        enable: true,
        listen: '0.0.0.0:53',
        nameserver: ['223.5.5.5', '119.29.29.29']
    };

    // 添加最终匹配规则
    config['rules'].push('MATCH,PROXY');

    log('配置自定义完成，共 ' + config.rules.length + ' 条规则');
}
```

### 说明

- 规则集 URL 支持缓存，相同 URL 在缓存期内不会重复下载
- 规则集并发下载，但会保持调用 `callback` 的顺序
- 不支持 ES6+ 的高级特性（goja 兼容 ES5.1）
- 不支持 `console.log`，请使用 `log()` 函数

## 项目结构

```
clash-converter/
├── main.go              # 入口函数
├── api_controller.go    # HTTP 路由和处理器
├── subscription.go      # 订阅解析和合并
├── config_builder.go    # 配置构建逻辑
├── js_runner.go         # JS 脚本执行引擎
├── dao.go               # 数据库操作
├── logger.go            # 日志系统
├── utils.go             # 工具函数
├── ui.html              # 前端界面
└── data/                # 数据目录（自动创建）
    └── database.db      # SQLite 数据库
```

## 许可证

MIT License