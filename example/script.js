// https://mihomo.party/docs/guide/override/javascript

// - User Config -
// -- Rules --

const NS_POLICY = {
    '+.fflogs.com': '119.29.29.29',
    '+.rpglogs.com': '119.29.29.29',
}

const DIRECT_PNAME = [
    'MonsterHunterWilds.exe',
    'SplitFiction.exe'
]

const DIRECT_SFX = [
    'fflogs.com',
    'rpglogs.com',
]

const PROXY_KW = [
    'jetbrains'
]

const PROXY_SFX = [
    'bobu.me',
    'linux.do',
    'sleazyfork.org',
    'fxacg.cc',
    'helloimg.com',
    'google.com',
]

const DIALERS = [
    {
        name: 'Dialer-1',
        type: 'ss',
        server: '1.14.5.14',
        port: 19198,
        password: 'password',
        cipher: 'aes-128-gcm',
        udp: true,
        'udp-over-tcp': true,
    },
]

const DIALER_GROUP_NAME = 'Dialer'
const DIALER_PROXY_GROUP_NAME = 'Dialer proxy'
const RELAY_GROUP_NAME = 'Relay'

// - Config Merger -

const RULES = [
    ...(DIRECT_PNAME.map(s => `PROCESS-NAME,${s},DIRECT`)),
    ...(DIRECT_SFX.map(s => `DOMAIN-SUFFIX,${s},DIRECT`)),
    ...(PROXY_KW.map(s => `DOMAIN-KEYWORD,${s},PROXY`)),
    ...(PROXY_SFX.map(s => `DOMAIN-SUFFIX,${s},PROXY`)),
]

const DIALER_NAMES = DIALERS.map(d => d.name)
DIALERS.forEach(d => d['dialer-proxy'] = DIALER_PROXY_GROUP_NAME)

// impl
function rulesets(r) {
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/LocalAreaNetwork.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/UnBan.list")
    r("WebAD", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanAD.list")
    r("AppAD", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/BanProgramAD.list")
    r("GoogleCN", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/GoogleFCM.list")
    r("GoogleCN", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/GoogleCN.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/SteamCN.list")
    r("Bing", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Bing.list")
    r("OneDrive", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/OneDrive.list")
    r("Microsoft", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Microsoft.list")
    r("Apple", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Apple.list")
    r("Telegram", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Telegram.list")
    r("OpenAI", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/OpenAi.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Epic.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Origin.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Sony.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Steam.list")
    r("Games", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Nintendo.list")
    r("YouTube", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/YouTube.list")
    r("Netflix", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Netflix.list")
    r("Bahamut", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Ruleset/Bahamut.list")
    r("ProxyMedia", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyMedia.list")
    r("PROXY", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ProxyGFWlist.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaDomain.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/ChinaCompanyIp.list")
    r("DIRECT", "https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/Download.list")
}

// impl
function buildConfig(config) {
    config['dns']['nameserver-policy'] = {
        ...config['dns']['nameserver-policy'],
        ...NS_POLICY
    }

    config['rules'] = [
        ...RULES,
        ...config['rules'],
        'GEOIP,CN,DIRECT',
        'MATCH,Other'
    ]

    const proxies = config['proxies']
    const proxy_names = proxies.map(p => p.name).filter(n => !/流量|到期/.test(n))
    config['proxies'] = [
        ...DIALERS,
        ...proxies,
    ]

    config['proxy-groups'] = buildGroups(proxy_names)

    return config
}

const SEL_GROUP = (name, proxies) => {
    return {
        name: name,
        type: 'select',
        proxies: proxies,
    }
}

function buildGroups(proxy_names) {
    const PROXIES = [ ...DIALER_NAMES, RELAY_GROUP_NAME, ...proxy_names ]
    const NORMAL =  [ 'PROXY', 'DIRECT', ...PROXIES ]
    const DEFAULT_DIRECT =  [ 'DIRECT', 'PROXY', ...PROXIES ]
    const REJECT = [ 'REJECT', 'DIRECT' ]

    return [
        SEL_GROUP(DIALER_GROUP_NAME, [ ...DIALER_NAMES, 'DIRECT' ]),
        SEL_GROUP(DIALER_PROXY_GROUP_NAME, [ ...proxy_names ]),
        SEL_GROUP('PROXY',  [ 'DIRECT', ...PROXIES ]),
        SEL_GROUP('OpenAI', NORMAL),
        SEL_GROUP('Telegram', NORMAL),
        SEL_GROUP('YouTube', NORMAL),
        SEL_GROUP('Netflix', NORMAL),
        SEL_GROUP('Bahamut', NORMAL),
        SEL_GROUP('ProxyMedia', NORMAL),
        SEL_GROUP('GoogleCN', NORMAL),
        SEL_GROUP('Bing', DEFAULT_DIRECT),
        SEL_GROUP('OneDrive', NORMAL),
        SEL_GROUP('Microsoft', DEFAULT_DIRECT),
        SEL_GROUP('Apple', DEFAULT_DIRECT),
        SEL_GROUP('Games', NORMAL),
        SEL_GROUP('WebAD', REJECT),
        SEL_GROUP('AppAD', REJECT),
        SEL_GROUP('Other', NORMAL),
        {
            name: RELAY_GROUP_NAME,
            type: 'relay',
            proxies: [
                DIALER_PROXY_GROUP_NAME,
                DIALER_GROUP_NAME
            ]
        },
    ]
}
