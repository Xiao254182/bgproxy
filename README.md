#### **使用场景**

在公司线上环境中，服务通过 Docker 容器运行 JAR 包部署，以下是典型的 Dockerfile：

```dockerfile
FROM openjdk:8-jre

ENV JAVA_OPTS="-server -Xms1024m -Xmx1024m -Dfile.encoding=utf-8"

WORKDIR /usr/share/service

ADD xxx.jar /usr/share/service/xxx.jar

ENTRYPOINT java -noverify ${JAVA_OPTS} -jar /usr/share/service/xxx.jar --server.port=8080
```

目前的服务更新方式为：

```bash
docker cp xxx-new.jar xxx-server:/usr/share/service/xxx.jar && docker restart xxx-server
```

**问题**：在容器重启期间，服务不可用，导致中断。

**现有解决方案：蓝绿发布**

1. 创建备用容器，运行相同服务。
2. 配置 Nginx 进行负载均衡（主容器处理流量，备用容器处于备用状态）。
3. 更新服务时：
    - 更新并启动备用容器。
    - 流量切换到备用容器。
    - 更新主容器，完成后将流量切回主容器。

**痛点**：

- 手动操作繁琐，特别是在大规模集群中。
- Nginx 配置复杂且冗余，难以维护。

为此，我们希望将反向代理功能直接集成到容器中，取代宿主机操作。

------

#### **解决方案优势**

1. **独立隔离**
    - 反向代理随容器运行，与宿主机解耦，提升服务独立性和灵活性。
2. **简化运维**
    - 容器内置代理，减少宿主机依赖，避免繁琐的 Nginx 配置管理。
3. **无中断更新**
    - 流量切换平滑，确保服务高可用。
4. **提升可移植性**
    - 反向代理与服务打包在同一镜像中，跨环境部署无需调整配置。
5. **支持自动化扩展**
    - 适配 CI/CD 流水线，轻松管理大规模集群。

通过这种优化，服务更新过程更加高效、稳定，完全满足企业级高可用场景需求。