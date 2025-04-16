window.serviceApp = function () {
    return {
        toast: {
            visible: false,
            message: '',
        },
        progress: 0,
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

        uploadJar() {
            const fileInput = document.getElementById('jarFile');
            if (!fileInput.files.length) {
                this.showToast("请选择文件");
                return;
            }

            const formData = new FormData();
            formData.append("jar", fileInput.files[0]);

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
