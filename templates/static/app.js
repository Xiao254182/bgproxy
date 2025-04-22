window.serviceApp = function () {
    return {
        toast: {
            visible: false,
            message: '',
        },
        progress: 0,
        dragOver: false, // 用于控制拖拽样式
        uploadedFile: null, // 存储上传的文件信息
        logModal: {
            visible: false,
            title: '',
            service: 'current',
            source: null,
            isFullLog: false,
        },

        scrollToBottom() {
            requestAnimationFrame(() => {
                const container = this.$refs.logContainer;
                container.scrollTop = container.scrollHeight;
            });
        },

        showToast(msg) {
            this.toast.message = msg;
            this.toast.visible = true;
            setTimeout(() => this.toast.visible = false, 3000);
        },

        handleDragOver() {
            this.dragOver = true; // 当拖拽区域上方时
        },

        handleDragLeave() {
            this.dragOver = false; // 当拖拽离开时
        },

        handleDrop(event) {
            this.dragOver = false; // 拖拽结束
            const file = event.dataTransfer.files[0];
            if (file) {
                this.uploadedFile = file;
                this.uploadJar();
            }
        },

        handleFileSelect(event) {
            const file = event.target.files[0];
            if (file) {
                this.uploadedFile = file;
                this.uploadJar();
            }
        },

        removeFile() {
            this.uploadedFile = null;
            document.getElementById('jarFile').value = ''; // 清空文件选择框
        },

        uploadJar() {
            if (!this.uploadedFile) {
                this.showToast("请选择文件");
                return;
            }

            const formData = new FormData();
            formData.append("jar", this.uploadedFile);

            const xhr = new XMLHttpRequest();
            xhr.open("POST", "/upload", true);

            xhr.upload.onprogress = (e) => {
                if (e.lengthComputable) {
                    this.progress = Math.round((e.loaded / e.total) * 100);
                }
            };

            xhr.onload = () => {
                if (xhr.status === 200) {
                    this.progress = 100;
                    this.showToast("上传成功 ✅");

                    // 解决闪一下问题：刷新前清除状态
                    this.uploadedFile = null;
                    this.progress = 0;

                    setTimeout(() => location.reload(), 1500);
                } else {
                    this.showToast("上传失败 ❌");
                }
            };

            xhr.send(formData);
        },

        switchService() {
            fetch('/switch', { method: 'POST' })
                .then(resp => {
                    if (resp.ok) {
                        this.showToast("切换成功 ✅");
                        setTimeout(() => location.reload(), 1500);
                    } else {
                        this.showToast("切换失败 ❌");
                    }
                });
        },

        openLogModal(service) {
            console.trace(`🔥 openLogModal(${service}) 被调用了`);
            this.logModal.title = service === 'current' ? '当前服务日志' : '新服务日志';
            this.logModal.service = service;
            this.logModal.visible = true;
            this.logModal.isFullLog = false;
            this.loadLog(service, false);
        },

        toggleLogMode() {
            this.logModal.isFullLog = !this.logModal.isFullLog;
            this.loadLog(this.logModal.service, this.logModal.isFullLog);
        },

        loadLog(service, full) {
            const el = this.$refs.logContent;
            el.textContent = '';

            if (this.logModal.source) {
                this.logModal.source.close();
            }

            const url = `/stream-log/${service}` + (full ? '?full=1' : '');
            this.logModal.source = new EventSource(url);
            this.logModal.source.onmessage = (event) => {
                el.textContent += event.data + '\n';
                this.scrollToBottom();
            };

            setTimeout(() => this.scrollToBottom(), 50);
        },

        closeLogModal() {
            if (this.logModal.source) {
                this.logModal.source.close();
                this.logModal.source = null;
            }
            this.logModal.visible = false;
        }
    }
}
