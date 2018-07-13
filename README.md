# 说明
基于 GIT 的简易代码部署工具

程序根据您提供的参数解析 git 命令返回结果，获得需要更新的文件，并利用 FTP 自动更新，具体的使用参数可以使用 ./deploy -h 参数获取。

# 使用方法
Like Unix:
```
./deploy -p=YOU-PROJECT-NAME-IN-CONF.JSON-FILE -b=master -n=20
```

Windows
```
deploy.exe -p=YOU-PROJECT-NAME-IN-CONF.JSON-FILE -b=master -n=20
```

  -h
        获取帮助文档
        
  -b string
        分支名称 (默认 "master" 分支)
        
  -n int
    `   要拉取的数据条数 (默认 20 条)
    
  -p string
        处理的项目名称
        
  -t string
        tag 名
