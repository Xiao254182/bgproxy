#!/bin/sh

# 启动流量控制器
/usr/local/bin/traffic-controller \
    -shadow=true \
    -jar /usr/share/service/app.jar &

# 等待控制器启动
sleep 2

# 输出日志到标准输出
tail -f /var/log/app/application.log