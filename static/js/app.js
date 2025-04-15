// 应用逻辑
$(document).ready(function() {
    // 初始化图表
    const chart = initChart();

    // 自动刷新数据
    setInterval(() => updateChart(chart), 5000);

    // 部署表单提交
    $('form').on('submit', function(e) {
        e.preventDefault();
        submitDeployForm(this);
    });
});

function initChart() {
    const chart = echarts.init(document.getElementById('requestChart'));
    const option = {
        title: { text: '请求趋势' },
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'time' },
        yAxis: { type: 'value' },
        series: [{
            name: '请求量',
            type: 'line',
            smooth: true,
            data: []
        }]
    };
    chart.setOption(option);
    return chart;
}

function updateChart(chart) {
    fetch('/status')
        .then(res => res.json())
        .then(data => {
            const option = chart.getOption();
            option.series[0].data.push({
                name: new Date().toLocaleTimeString(),
                value: [new Date(), data.requests]
            });
            chart.setOption(option);
        });
}

function submitDeployForm(form) {
    const formData = new FormData(form);

    fetch('/deploy', {
        method: 'POST',
        body: formData,
        headers: {
            'X-API-Key': localStorage.getItem('apiKey')
        }
    })
        .then(response => {
            if (response.ok) {
                alert('部署已启动！');
                location.reload();
            }
        })
        .catch(error => {
            console.error('Error:', error);
        });
}

window.performRollback = function() {
    if (confirm('确认要回滚到备用版本吗？')) {
        fetch('/rollback', {
            method: 'POST',
            headers: {
                'X-API-Key': localStorage.getItem('apiKey')
            }
        })
            .then(response => {
                if (response.ok) {
                    location.reload();
                }
            });
    }
}