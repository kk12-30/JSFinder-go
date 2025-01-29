# JSFinder-go

![JSFinder-go](https://img.shields.io/badge/JSFinder-go-blue.svg)

JSFinder-go 是一个基于 Go 语言开发的工具，用于从网站的 JavaScript 文件中快速提取 URL。

## 📌 项目地址
[https://github.com/kk12-30/JSFinder-go](https://github.com/kk12-30/JSFinder-go)


## 💡 使用方法
### 运行基本命令
```sh
js.exe -u https://example.com
```
该命令会扫描 `https://example.com` 站点的 JavaScript 文件，并提取 URL。

### 参数说明
| 参数 | 说明 |
|------|------|
| `-u` | 目标网站地址 |
| `-c` | 指定目标网站的 Cookie（可用于访问需要身份验证的页面） |
| `-f` | 指定包含 URL 或 JS 文件的路径，用于批量处理 |
| `-ou` | 提取的 URL 输出文件名（默认为 `url.txt`） |
| `-a` | 提取并处理后的 URL 保存到 `url.txt` 文件中 |
| `-t` | 指定保留的路径层级数（-1 表示完整路径） |

### 批量处理多个 URL 或 JS 文件
```sh
js.exe -f targets.txt
```
`targets.txt` 文件的内容可以是 URL 或 JS 文件路径，每行一个。

### 保存提取的 URL
使用 `-a` 参数可以将提取的 URL 保存到 `url.txt` 文件中。
```sh
js.exe -u https://example.com -a
```

## 📊 运行示例
```sh
js.exe -u https://example.com
```
示例输出：
```
提取结果:
https://example.com/api/data.json    [Size: 2.3 KB]
https://example.com/static/script.js    [Size: 1.5 KB]
```

## 🔍 过滤和去重机制
JSFinder-go 采用以下机制来保证提取 URL 的准确性：
- 仅提取符合正则表达式规则的 URL。
- 去除重复的 URL，避免冗余数据。
- 仅保留与目标站点相关的 URL，避免无关内容干扰。
- 对 JavaScript 文件中的 URL 进行解析，自动补全相对路径。

## 📂 输出文件格式
当使用 `-a` 参数时，提取的 URL 将会保存到 `url.txt` 文件中，文件内容示例如下：
```
https://example.com/api/data.json
https://example.com/static/script.js
```

## ❤️ 原项目
https://github.com/Threezh1/JSFinder
