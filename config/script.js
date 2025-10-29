// https://mihomo.party/docs/guide/override/javascript

// ========== 这个文件是用来给 Clash 配置合并/扩展的脚本 ==========
// 作用简述（小白版）:
//  这个脚本定义了一些自定义规则、代理（dialer）和代理组的模板，然后在
//  将这些配置合并进传入的 `config` 中。它通常被用来自动化生成 Clash 的配置
// （例如把用户的代理列表、规则集合和 DNS 策略拼接成最终的配置文件）。
// 主要输出/效果：
//  - 修改 `config.dns.nameserver-policy`，合并自定义 DNS 策略
//  - 在 `config.rules` 前面插入自定义规则（比如进程名直连、特定域直连或走代理）
//  - 在 `config.proxies` 前端插入自定义的拨号器（DIALERS）
//  - 生成并替换 `config['proxy-groups']`（代理组）以包含拨号器与转发组

// - User Config -
// 以下为可按需修改的“用户配置”部分：

// DNS 名称服务器策略（示例）
// 说明：把访问某些域名时使用的 DNS 服务器指定为特定 IP（常用于自定义解析）
const NS_POLICY = {
    // '+.fflogs.com' 表示所有以 fflogs.com 结尾的域名
    // '+.fflogs.com': '119.29.29.29',
    // '+.rpglogs.com': '119.29.29.29',
}

// 下面这些数组用于生成规则列表：
// DIRECT_PNAME：指定某些进程名走直连（不经过代理），例如本地游戏客户端
const DIRECT_PNAME = [
    'MonsterHunterWilds.exe',
    'SplitFiction.exe',
    'OneDrive.exe',
    'OneDrive.Sync.Service.exe',
]

// DIRECT_SFX：指定以这些后缀结尾的域名走直连
const DIRECT_SFX = [
    'fflogs.com',
    'rpglogs.com',
    'edge.microsoft.com',//微软Edge浏览器更新域名
]

// DIRECT_KW：如果域名中包含这些关键字，则走直连
const DIRECT_KW = [

    'clash.razord.top',//这俩是Clash的网页管理页面
    'yacd.haishan.me',

]

// PROXY_KW：如果域名中包含这些关键字，则走代理
const PROXY_KW = [
    'jetbrains',
    'ddys',
    'msftconnecttest',//校园网检测

]

// PROXY_SFX：域名后缀匹配列表，匹配则走代理
const PROXY_SFX = [
    'bobu.me',
    'linux.do',
    'sleazyfork.org',
    'fxacg.cc',
    'helloimg.com',
    'googleapis.com',//谷歌翻译
    'copilot.microsoft.com',//Copilot
    'copilot.cloud.microsoft.com',//Copilot
    'hdhive.online',

    // 'google.com',

]

// DIALERS：自定义的拨号器 / 代理节点（示例）
// 说明：这是一个代理配置数组，会被插入到最终的 `config.proxies` 前面。
const DIALERS = [
    // {
    //     name: 'Dialer-1',
    //     type: 'ss',
    //     server: '1.14.5.14',
    //     port: 19198,
    //     password: 'password',
    //     cipher: 'aes-128-gcm',
    //     udp: true,
    //     'udp-over-tcp': true,
    // },
]

// 代理组名称常量（后面会用到）
const DIALER_GROUP_NAME = 'Dialer'
const DIALER_PROXY_GROUP_NAME = 'Dialer proxy'
const RELAY_GROUP_NAME = 'Relay'

// 自定义代理组规则（按服务分类，指定哪些节点走这个组）
const GEMINI_NODES = '日本|美国|JP|US'  // Gemini 用日本或美国节点
const OPENAI_NODES = '日本|新加坡|SG'   // OpenAI 用日本或新加坡节点
const YOUTUBE_NODES = '美国|香港|HK'    // YouTube 用美国或香港节点

// ========== 自定义规则集 ==========
// 格式：{ name: '规则集名称', domains: ['域名1', '域名2'], nodes: '节点正则表达式' }
// 说明：
//   - name: 规则集的名称（也是代理组的名称）
//   - domains: 要匹配的域名列表（支持后缀匹配）
//   - nodes: 指定哪些节点走这个规则（用正则表达式，如 '日本|美国|JP|US'）
const MY_RULES = [
    {
        name: 'myRule',
        domains_suffix: [
           
            'bard.google.com',
            'deepmind.com',
            'deepmind.google',
            'gemini.google.com',
            'generativeai.google',
            'proactivebackend-pa.googleapis.com',
            'apis.google.com',
  
            // 在这里继续添加你想要的域名
        ],
        domains_keyword: [
            'colab',
            'developerprofiles',
            'generativelanguage',
            'pa.google',

        ],
        domains: [
            'aistudio.google.com',
            'ai.google.dev',
            'alkalimakersuite-pa.clients6.google.com',
            'makersuite.google.com',
            'generativelanguage.googleapis.com',
            'music.youtube.com',

        ],
        nodes: '日本|美国|JP|US|新加坡|台湾'  // 改成你想要的节点
    },
    // 可以继续添加更多自定义规则
    // {
    //     name: 'myRule2',
    //     domains: ['site1.com', 'site2.com'],
    //     nodes: '新加坡|SG'
    // },
]

// - Config Merger -

// RULES：把上面的数组转换成 Clash 规则字符串（例如 PROCESS-NAME,xxx,DIRECT）
const RULES = [
    ...(DIRECT_PNAME.map(s => `PROCESS-NAME,${s},DIRECT`)),
    ...(DIRECT_SFX.map(s => `DOMAIN-SUFFIX,${s},DIRECT`)),
    ...(PROXY_KW.map(s => `DOMAIN-KEYWORD,${s},PROXY`)),
    ...(PROXY_SFX.map(s => `DOMAIN-SUFFIX,${s},PROXY`)),
    ...(DIRECT_KW.map(s => `DOMAIN-KEYWORD,${s},DIRECT`)),
    // 正则表达式规则（匹配 Gemini 相关域名）
    // 'DOMAIN-REGEX,^.*gemini.*$,Gemini',
    // 'DOMAIN-REGEX,^.*generativeai.*$,Gemini',
    // 自定义规则集的域名规则
    ...(MY_RULES.flatMap(rule => [
        ...(rule.domains_suffix?.filter(domain => domain && domain.trim())?.map(domain => `DOMAIN-SUFFIX,${domain.trim()},${rule.name}`) || []),
        ...(rule.domains_keyword?.filter(domain => domain && domain.trim())?.map(domain => `DOMAIN-KEYWORD,${domain.trim()},${rule.name}`) || []),
        ...(rule.domains?.filter(domain => domain && domain.trim())?.map(domain => `DOMAIN,${domain.trim()},${rule.name}`) || [])
    ])),
    // Remote ruleset -> proxy-group mappings (rule-provider use)
    // 'RULE-SET,Gemini,OpenAI',
    // 'RULE-SET,BardAI,OpenAI',
    // 'RULE-SET,Chromecast,YouTube',
    // 'RULE-SET,YouTubeMusic,YouTube',
]

// 获取所有拨号器名称（用于构造代理组）
const DIALER_NAMES = DIALERS.map(d => d.name)
// 给每个拨号器对象加上 'dialer-proxy' 字段，值为拨号器代理组名（后面的代理组会用到）
DIALERS.forEach(d => d['dialer-proxy'] = DIALER_PROXY_GROUP_NAME)

// impl
// rulesets 函数：注册一组远程规则集（按名称 + URL）
// 参数 r 是一个“注册器”函数，调用 r(name, url) 会把远程规则加入到最终配置中
function rulesets(r) {
    //本地局域网
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list")
    //不被墙网站
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/UnBan.list")
    //广告屏蔽
    r("WebAD", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanAD.list")
    r("AppAD", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanProgramAD.list")
    r("GoogleCN", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/GoogleFCM.list")
    r("GoogleCN", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/GoogleCN.list")
    //Steam中国
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/SteamCN.list")
    //微软服务
    r("Bing", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Bing.list")
    r("OneDrive", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/OneDrive.list")
    r("Microsoft", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Microsoft.list")
    //苹果服务
    r("Apple", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Apple.list")
    //Telegram
    r("Telegram", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Telegram.list")
    //OpenAI
    r("OpenAI", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/OpenAi.list")
    //Google
    r("Google", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Google.list")
    // r("Gemini", "https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/Gemini/Gemini.yaml")
    // r("BardAI","https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/BardAI/BardAI.yaml")    
    // r("Chromecast","https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/Chromecast/Chromecast.yaml") 
    // r("YouTubeMusic","https://raw.githubusercontent.com/blackmatrix7/ios_rule_script/master/rule/Clash/YouTubeMusic/YouTubeMusic.yaml") 


    //游戏服务
    //Epic/Origin/Sony/Steam/任天堂
    // r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Epic.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Origin.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Sony.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Steam.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Nintendo.list")
    //流媒体服务
    //YouTube/Netflix/Bahamut/ProxyMedia
    r("YouTube", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/YouTube.list")
    r("Netflix", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Netflix.list")
    r("Bahamut", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Bahamut.list")
    r("ProxyMedia", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyMedia.list")
    //代理规则
    r("PROXY", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyGFWlist.list")
    //国内规则
    // ChinaDomain/ChinaCompanyIp/Download
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaCompanyIp.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Download.list")
}

// impl


function buildConfig(config) {
    // 仅在用户实际配置了 NS_POLICY 时才合并，避免意外覆盖或引入不可达的 nameserver
    if (NS_POLICY && Object.keys(NS_POLICY).length > 0) {
        config['dns']['nameserver-policy'] = {
            ...config['dns']['nameserver-policy'],
            ...NS_POLICY
        }
    }

    config['rules'] = [
        ...RULES,
        ...config['rules'],
        'GEOIP,CN,DIRECT',
        'MATCH,Other'
    ]

    const proxies = config['proxies']
    const proxy_names = proxies.map(p => p.name).filter(n => !/流量|到期/.test(n))
    // 仅在存在拨号器时把它们插入到 proxies 前面；否则保留上游 proxies 顺序不变
    if (Array.isArray(DIALERS) && DIALERS.length > 0) {
        config['proxies'] = [
            ...DIALERS,
            ...proxies,
        ]
    } else {
        config['proxies'] = proxies
    }

    config['proxy-groups'] = buildGroups(proxy_names)
}

const SEL_GROUP = (name, proxies) => {
    return {
        name: name,
        type: 'select',
        proxies: proxies,
    }
}

// 过滤函数：根据正则表达式从代理列表中过滤出匹配的节点
const filterProxiesByRegex = (proxy_names, regex) => {
    const pattern = new RegExp(regex)
    return proxy_names.filter(name => pattern.test(name))
}

function buildGroups(proxy_names) {
    // 当没有拨号器时，避免生成 Dialer / Relay 等占位组
    const hasDialers = Array.isArray(DIALER_NAMES) && DIALER_NAMES.length > 0

    const PROXIES = hasDialers ? [ ...DIALER_NAMES, RELAY_GROUP_NAME, ...proxy_names ] : [ ...proxy_names ]
    const NORMAL =  [ 'PROXY', 'DIRECT', ...PROXIES ]
    const DEFAULT_DIRECT =  [ 'DIRECT', 'PROXY', ...PROXIES ]
    const REJECT = [ 'REJECT', 'DIRECT' ]

    // 根据正则表达式过滤节点
    const gemini_nodes = filterProxiesByRegex(proxy_names, GEMINI_NODES)
    const openai_nodes = filterProxiesByRegex(proxy_names, OPENAI_NODES)
    const youtube_nodes = filterProxiesByRegex(proxy_names, YOUTUBE_NODES)

    // 生成自定义规则集的代理组
    const my_rule_groups = MY_RULES.map(rule => 
        SEL_GROUP(rule.name, [ ...filterProxiesByRegex(proxy_names, rule.nodes), 'DIRECT' ])
    )

    const groups = []

    if (hasDialers) {
        groups.push(SEL_GROUP(DIALER_GROUP_NAME, [ ...DIALER_NAMES, 'DIRECT' ]))
        groups.push(SEL_GROUP(DIALER_PROXY_GROUP_NAME, [ ...proxy_names ]))
    }

    groups.push(SEL_GROUP('PROXY',  [ 'DIRECT', ...PROXIES ]))
    // Gemini 代理组：使用正则表达式自动过滤日本或美国节点
    // groups.push(SEL_GROUP('Gemini', [ ...gemini_nodes, 'DIRECT' ]))
    groups.push(SEL_GROUP('OpenAI', NORMAL))
    // 自定义规则集的代理组
    groups.push(...my_rule_groups)
    groups.push(SEL_GROUP('Telegram', NORMAL))
    groups.push(SEL_GROUP('YouTube', NORMAL))
    groups.push(SEL_GROUP('Netflix', NORMAL))
    groups.push(SEL_GROUP('Bahamut', NORMAL))
    groups.push(SEL_GROUP('ProxyMedia', NORMAL))
    groups.push(SEL_GROUP('GoogleCN', NORMAL))
    groups.push(SEL_GROUP('Bing', DEFAULT_DIRECT))
    groups.push(SEL_GROUP('OneDrive', NORMAL))
    groups.push(SEL_GROUP('Microsoft', DEFAULT_DIRECT))
    groups.push(SEL_GROUP('Google', NORMAL))
    groups.push(SEL_GROUP('Apple', DEFAULT_DIRECT))
    groups.push(SEL_GROUP('Games', NORMAL))
    groups.push(SEL_GROUP('WebAD', REJECT))
    groups.push(SEL_GROUP('AppAD', REJECT))
    groups.push(SEL_GROUP('Other', NORMAL))

    if (hasDialers) {
        groups.push({
            name: RELAY_GROUP_NAME,
            type: 'relay',
            proxies: [
                DIALER_PROXY_GROUP_NAME,
                DIALER_GROUP_NAME
            ]
        })
    }

    return groups
}
