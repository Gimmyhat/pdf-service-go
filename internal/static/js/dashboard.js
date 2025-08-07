// Dashboard JS - объединенный функционал для статистики и ошибок
let statsData = null;
let errorData = null;
let charts = {};

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    initializeDashboard();
    setupTabHandlers();
    refreshAllData();
    
    // Автообновление каждые 30 секунд
    setInterval(refreshAllData, 30000);
});

function initializeDashboard() {
    updateLastUpdateTime();
    console.log('Dashboard initialized');
}

function setupTabHandlers() {
    // Обработчики переключения табов
    const tabTriggerList = [].slice.call(document.querySelectorAll('#dashboardTabs button[data-bs-toggle="tab"]'));
    tabTriggerList.forEach(function(tabTrigger) {
        tabTrigger.addEventListener('shown.bs.tab', function(event) {
            const targetTab = event.target.getAttribute('data-bs-target');
            
            // Обновляем данные при переключении на таб
            switch(targetTab) {
                case '#statistics':
                    if (!statsData) {
                        updateStats();
                    }
                    break;
                case '#errors':
                    if (!errorData) {
                        refreshErrorData();
                    }
                    break;
                case '#overview':
                    updateOverview();
                    break;
            }
        });
    });
}

// Универсальная функция обновления всех данных
function refreshAllData() {
    updateLastUpdateTime();
    loadSystemHealth();
    updateQuickMetrics();
    
    // Обновляем активный таб
    const activeTab = document.querySelector('#dashboardTabs .nav-link.active');
    const activeTarget = activeTab?.getAttribute('data-bs-target');
    
    switch(activeTarget) {
        case '#statistics':
            updateStats();
            break;
        case '#errors':
            refreshErrorData();
            break;
        case '#overview':
        default:
            updateOverview();
            break;
    }
}

// Системное здоровье
async function loadSystemHealth() {
    try {
        const response = await fetch('/health');
        const health = await response.json();
        
        const healthIndicator = document.querySelector('#systemHealth .health-indicator');
        const healthText = document.getElementById('healthText');
        
        if (health.status === 'healthy') {
            healthIndicator.className = 'health-indicator health-healthy';
            healthText.textContent = 'Healthy';
        } else {
            healthIndicator.className = 'health-indicator health-critical';
            healthText.textContent = 'Unhealthy';
        }
        
        // Обновляем здоровье компонентов
        updateComponentHealth(health.details);
        
    } catch (error) {
        console.error('Error loading system health:', error);
        const healthIndicator = document.querySelector('#systemHealth .health-indicator');
        const healthText = document.getElementById('healthText');
        healthIndicator.className = 'health-indicator health-critical';
        healthText.textContent = 'Error';
    }
}

function updateComponentHealth(details) {
    if (!details || !details.circuit_breakers) return;
    
    const componentHealth = document.getElementById('componentHealth');
    if (!componentHealth) return;
    
    const components = [
        { name: 'Gotenberg', key: 'gotenberg' },
        { name: 'DOCX Generator', key: 'docx_generator' },
        { name: 'Database', key: 'database' },
        { name: 'Error Tracking', key: 'error_tracking' }
    ];
    
    let html = '';
    components.forEach(comp => {
        const status = details.circuit_breakers[comp.key]?.status || true;
        const healthClass = status ? 'health-healthy' : 'health-critical';
        
        html += `
            <div class="d-flex justify-content-between align-items-center mb-2">
                <span>${comp.name}</span>
                <span class="health-indicator ${healthClass}"></span>
            </div>
        `;
    });
    
    componentHealth.innerHTML = html;
}

// Быстрые метрики в заголовке
async function updateQuickMetrics() {
    try {
        // Загружаем статистику
        const statsResponse = await fetch('/api/v1/statistics?period=24hours');
        const stats = await statsResponse.json();
        
        // Загружаем ошибки
        const errorsResponse = await fetch('/api/v1/errors/stats?period=24h');
        const errors = await errorsResponse.json();
        
        // Обновляем метрики
        const totalRequests = stats.requests?.total || 0;
        const successfulRequests = stats.requests?.success || 0;
        const successRate = totalRequests > 0 ? Math.round((successfulRequests / totalRequests) * 100) : 100;
        
        document.getElementById('totalRequests').textContent = formatNumber(totalRequests);
        document.getElementById('successRate').textContent = successRate + '%';
        document.getElementById('avgResponseTime').textContent = 
            stats.docx?.average_duration || '--';
        document.getElementById('totalErrors').textContent = formatNumber(errors.errors_24h || 0);
        
    } catch (error) {
        console.error('Error updating quick metrics:', error);
    }
}

// Обзорная страница
async function updateOverview() {
    try {
        // Обновляем последние события
        await updateRecentEvents();
        
        // Создаем обзорный график с реальными данными
        await createOverviewChart();
        
    } catch (error) {
        console.error('Error updating overview:', error);
    }
}

async function updateRecentEvents() {
    try {
        const response = await fetch('/api/v1/errors?limit=5');
        const data = await response.json();
        const events = data.summary?.recent_errors || [];
        
        const container = document.getElementById('recentEvents');
        if (events.length === 0) {
            container.innerHTML = '<small class="text-muted">Нет недавних событий</small>';
            return;
        }
        
        let html = '';
        events.forEach(event => {
            const timeAgo = new Date(event.timestamp).toLocaleString('ru');
            const severityClass = getSeverityClass(event.severity);
            
            html += `
                <div class="border-bottom pb-2 mb-2">
                    <div class="d-flex justify-content-between align-items-center">
                        <small class="fw-bold text-${severityClass}">${event.component || 'System'}</small>
                        <small class="text-muted">${timeAgo}</small>
                    </div>
                    <small class="text-muted">${event.message?.substring(0, 60)}...</small>
                </div>
            `;
        });
        
        container.innerHTML = html;
        
    } catch (error) {
        console.error('Error updating recent events:', error);
        document.getElementById('recentEvents').innerHTML = 
            '<small class="text-danger">Ошибка загрузки событий</small>';
    }
}

async function createOverviewChart() {
    const ctx = document.getElementById('overviewChart');
    if (!ctx) return;
    
    // Уничтожаем существующий график
    if (charts.overview) {
        charts.overview.destroy();
    }
    
    try {
        // Получаем реальные данные от API
        const [statsResponse, errorsResponse] = await Promise.all([
            fetch('/api/v1/statistics?period=24hours'),
            fetch('/api/v1/errors/stats?period=24h')
        ]);
        
        const stats = await statsResponse.json();
        const errors = await errorsResponse.json();
        
        const now = new Date();
        const labels = [];
        const requestsData = [];
        const errorsData = [];
        
        // Используем реальные данные по часам из API
        const apiHourData = stats.requests?.by_hour_of_day || {};
        
        for (let i = 23; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 60 * 60 * 1000);
            const hour = time.getHours().toString().padStart(2, '0');
            
            labels.push(time.toLocaleTimeString('ru', { hour: '2-digit' }));
            
            // Используем реальные данные или 0, если данных нет
            const hourRequests = apiHourData[hour] || 0;
            requestsData.push(hourRequests);
            
            // Пока нет почасовых данных по ошибкам, используем пропорцию
            const errorRate = stats.requests?.total > 0 ? 
                (stats.requests.total - stats.requests.success) / stats.requests.total : 0;
            errorsData.push(Math.round(hourRequests * errorRate));
        }
        
        charts.overview = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Запросы',
                    data: requestsData,
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.1)',
                    tension: 0.4,
                    yAxisID: 'y'
                }, {
                    label: 'Ошибки',
                    data: errorsData,
                    borderColor: 'rgb(255, 99, 132)',
                    backgroundColor: 'rgba(255, 99, 132, 0.1)',
                    tension: 0.4,
                    yAxisID: 'y1'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                scales: {
                    x: {
                        display: true,
                        title: {
                            display: true,
                            text: 'Время (часы)'
                        }
                    },
                    y: {
                        type: 'linear',
                        display: true,
                        position: 'left',
                        title: {
                            display: true,
                            text: 'Запросы'
                        }
                    },
                    y1: {
                        type: 'linear',
                        display: true,
                        position: 'right',
                        title: {
                            display: true,
                            text: 'Ошибки'
                        },
                        grid: {
                            drawOnChartArea: false,
                        },
                    }
                },
                plugins: {
                    title: {
                        display: true,
                        text: 'Тренды за последние 24 часа'
                    },
                    legend: {
                        display: true,
                        position: 'top'
                    }
                }
            }
        });
        
    } catch (error) {
        console.error('Error creating overview chart:', error);
        
        // В случае ошибки показываем пустой график с сообщением
        const now = new Date();
        const labels = [];
        for (let i = 23; i >= 0; i--) {
            const time = new Date(now.getTime() - i * 60 * 60 * 1000);
            labels.push(time.toLocaleTimeString('ru', { hour: '2-digit' }));
        }
        
        charts.overview = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Нет данных',
                    data: Array(24).fill(0),
                    borderColor: 'rgb(200, 200, 200)',
                    backgroundColor: 'rgba(200, 200, 200, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    title: {
                        display: true,
                        text: 'Ошибка загрузки данных трендов'
                    }
                }
            }
        });
    }
}

// Статистика (адаптировано из charts.js)
async function updateStats() {
    const period = document.getElementById('statsPeriodSelect')?.value || 'all';
    
    try {
        const response = await fetch(`/api/v1/statistics?period=${period}`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        statsData = await response.json();
        displayStats(statsData);
        createStatCharts(statsData);
        
    } catch (error) {
        console.error('Error loading statistics:', error);
        displayErrorMessage('Ошибка загрузки статистики');
    }
}

function displayStats(data) {
    const container = document.getElementById('statsContainer');
    if (!container) return;
    
    if (!data) {
        container.innerHTML = '<div class="alert alert-warning">Нет данных для отображения</div>';
        return;
    }
    
    // Адаптируем структуру данных API к ожидаемому формату
    const totalRequests = data.requests?.total || 0;
    const successfulRequests = data.requests?.success || 0;
    const docxErrors = data.docx?.error_generations || 0;
    const gotenbergErrors = data.gotenberg?.error_requests || 0;
    
    const stats = [
        { label: 'Всего запросов', value: formatNumber(totalRequests) },
        { label: 'Успешных запросов', value: formatNumber(successfulRequests) },
        { label: 'Ошибок DOCX', value: formatNumber(docxErrors) },
        { label: 'Ошибок Gotenberg', value: formatNumber(gotenbergErrors) },
        { label: 'Среднее время генерации', value: data.docx?.average_duration || 'N/A' },
        { label: 'Средний размер PDF', value: data.pdf?.average_size || 'N/A' },
        { label: 'Макс. время генерации', value: data.docx?.max_duration || 'N/A' },
        { label: 'Макс. размер PDF', value: data.pdf?.max_size || 'N/A' }
    ];
    
    let html = '';
    stats.forEach(stat => {
        html += `
            <div class="stat-item">
                <div class="stat-label">${stat.label}</div>
                <div class="stat-value">${stat.value}</div>
            </div>
        `;
    });
    
    container.innerHTML = html;
}

function createStatCharts(data) {
    // Создаем графики статистики (адаптированные из оригинального charts.js)
    createWeekdayChart(data);
    createHourChart(data);
    createDocxChart(data);
    createGotenbergChart(data);
    createPdfSizeChart(data);
}

// Адаптированные функции создания графиков из charts.js
function createWeekdayChart(data) {
    const ctx = document.getElementById('weekdayChart');
    if (!ctx) return;
    
    if (charts.weekday) {
        charts.weekday.destroy();
    }
    
    const weekdays = ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье'];
    
    // Преобразуем данные API в нужный формат
    const apiWeekData = data.requests?.by_day_of_week || {};
    const weekdayMapping = {
        'Monday': 0, 'Tuesday': 1, 'Wednesday': 2, 'Thursday': 3,
        'Friday': 4, 'Saturday': 5, 'Sunday': 6
    };
    
    const weekdayData = Array(7).fill(0);
    Object.keys(apiWeekData).forEach(day => {
        const index = weekdayMapping[day];
        if (index !== undefined) {
            weekdayData[index] = apiWeekData[day];
        }
    });
    
    charts.weekday = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: weekdays,
            datasets: [{
                label: 'Запросы',
                data: weekdayData,
                backgroundColor: 'rgba(54, 162, 235, 0.5)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                }
            },
            scales: {
                y: {
                    beginAtZero: true
                }
            }
        }
    });
}

function createHourChart(data) {
    const ctx = document.getElementById('hourChart');
    if (!ctx) return;
    
    if (charts.hour) {
        charts.hour.destroy();
    }
    
    const hours = Array.from({length: 24}, (_, i) => `${i}:00`);
    
    // Преобразуем данные API в нужный формат
    const apiHourData = data.requests?.by_hour_of_day || {};
    const hourData = Array(24).fill(0);
    Object.keys(apiHourData).forEach(hour => {
        const hourIndex = parseInt(hour, 10);
        if (hourIndex >= 0 && hourIndex < 24) {
            hourData[hourIndex] = apiHourData[hour];
        }
    });
    
    charts.hour = new Chart(ctx, {
        type: 'line',
        data: {
            labels: hours,
            datasets: [{
                label: 'Запросы',
                data: hourData,
                borderColor: 'rgba(255, 99, 132, 1)',
                backgroundColor: 'rgba(255, 99, 132, 0.1)',
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                }
            },
            scales: {
                y: {
                    beginAtZero: true
                }
            }
        }
    });
}

function createDocxChart(data) {
    const ctx = document.getElementById('docxChart');
    if (!ctx) return;
    
    if (charts.docx) {
        charts.docx.destroy();
    }
    
    const successful = (data.docx?.total_generations || 0) - (data.docx?.error_generations || 0);
    const errors = data.docx?.error_generations || 0;
    
    charts.docx = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['Успешно', 'Ошибки'],
            datasets: [{
                data: [successful, errors],
                backgroundColor: ['rgba(75, 192, 192, 0.8)', 'rgba(255, 99, 132, 0.8)']
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false
        }
    });
}

function createGotenbergChart(data) {
    const ctx = document.getElementById('gotenbergChart');
    if (!ctx) return;
    
    if (charts.gotenberg) {
        charts.gotenberg.destroy();
    }
    
    const successful = (data.gotenberg?.total_requests || 0) - (data.gotenberg?.error_requests || 0);
    const errors = data.gotenberg?.error_requests || 0;
    
    charts.gotenberg = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['Успешно', 'Ошибки'],
            datasets: [{
                data: [successful, errors],
                backgroundColor: ['rgba(54, 162, 235, 0.8)', 'rgba(255, 206, 86, 0.8)']
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false
        }
    });
}

function createPdfSizeChart(data) {
    const ctx = document.getElementById('pdfSizeChart');
    if (!ctx) return;
    
    if (charts.pdfSize) {
        charts.pdfSize.destroy();
    }
    
    // Симулируем данные по размерам PDF
    const sizeData = [
        data.small_pdfs || 0,
        data.medium_pdfs || 0,
        data.large_pdfs || 0
    ];
    
    charts.pdfSize = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: ['< 1MB', '1-5MB', '> 5MB'],
            datasets: [{
                data: sizeData,
                backgroundColor: [
                    'rgba(153, 102, 255, 0.8)',
                    'rgba(255, 159, 64, 0.8)',
                    'rgba(255, 99, 132, 0.8)'
                ]
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false
        }
    });
}

// Ошибки (адаптировано из errors.js)
async function refreshErrorData() {
    const period = document.getElementById('errorPeriodFilter')?.value || '24h';
    const type = document.getElementById('errorTypeFilter')?.value || '';
    const component = document.getElementById('errorComponentFilter')?.value || '';
    const severity = document.getElementById('errorSeverityFilter')?.value || '';
    
    try {
        // Загружаем статистику ошибок
        const statsResponse = await fetch(`/api/v1/errors/stats?period=${period}`);
        const stats = await statsResponse.json();
        
        // Загружаем детальную информацию об ошибках
        let url = `/api/v1/errors?period=${period}&limit=50`;
        if (type) url += `&type=${type}`;
        if (component) url += `&component=${component}`;
        if (severity) url += `&severity=${severity}`;
        
        const errorsResponse = await fetch(url);
        const errors = await errorsResponse.json();
        
        errorData = { stats, errors };
        displayErrorStats(stats);
        displayErrorList(errors.summary?.recent_errors || []);
        
    } catch (error) {
        console.error('Error loading error data:', error);
        displayErrorMessage('Ошибка загрузки данных об ошибках');
    }
}

function displayErrorStats(stats) {
    document.getElementById('errorTotalErrors').textContent = formatNumber(stats.total_errors || 0);
    document.getElementById('errorErrors24h').textContent = formatNumber(stats.errors_24h || 0);
    document.getElementById('errorErrors1h').textContent = formatNumber(stats.errors_1h || 0);
}

function displayErrorList(errors) {
    const container = document.getElementById('errorRecentErrors');
    if (!container) return;
    
    if (errors.length === 0) {
        container.innerHTML = '<div class="alert alert-success">Нет ошибок за выбранный период</div>';
        return;
    }
    
    let html = '';
    errors.forEach(error => {
        const severityClass = getSeverityClass(error.severity);
        const timeAgo = new Date(error.timestamp).toLocaleString('ru');
        
        html += `
            <div class="error-card card ${error.severity}">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start mb-2">
                        <h6 class="card-title mb-0">
                            <span class="badge bg-${severityClass} error-badge">${error.severity?.toUpperCase()}</span>
                            ${error.component || 'Unknown Component'}
                        </h6>
                        <small class="text-muted">${timeAgo}</small>
                    </div>
                    
                    <p class="card-text">${error.message || 'No message available'}</p>
                    
                    <div class="row text-small">
                        <div class="col-sm-6">
                            <strong>Type:</strong> ${error.error_type || 'unknown'}
                        </div>
                        <div class="col-sm-6">
                            <strong>Request ID:</strong> ${error.request_id || 'N/A'}
                        </div>
                    </div>
                    
                    ${error.trace_id ? `
                        <div class="mt-2">
                            <a href="#" class="jaeger-link" onclick="openJaegerTrace('${error.trace_id}')">
                                <i class="bi bi-search"></i> View in Jaeger
                            </a>
                        </div>
                    ` : ''}
                    
                    ${error.stack_trace ? `
                        <div class="mt-2">
                            <button class="btn btn-sm btn-outline-secondary" type="button" 
                                    data-bs-toggle="collapse" data-bs-target="#stack-${error.request_id}" 
                                    aria-expanded="false">
                                Show Stack Trace
                            </button>
                            <div class="collapse mt-2" id="stack-${error.request_id}">
                                <div class="stack-trace">${error.stack_trace}</div>
                            </div>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;
    });
    
    container.innerHTML = html;
}

// Утилиты
function getSeverityClass(severity) {
    switch (severity?.toLowerCase()) {
        case 'critical': return 'danger';
        case 'high': return 'warning';
        case 'medium': return 'info';
        case 'low': return 'secondary';
        default: return 'secondary';
    }
}

function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

function formatDuration(seconds) {
    if (seconds < 1) {
        return Math.round(seconds * 1000) + 'ms';
    } else if (seconds < 60) {
        return seconds.toFixed(1) + 's';
    } else {
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = Math.round(seconds % 60);
        return `${minutes}m ${remainingSeconds}s`;
    }
}

function formatBytes(bytes) {
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    if (bytes === 0) return '0 Bytes';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
}

function updateLastUpdateTime() {
    const lastUpdate = document.getElementById('lastUpdate');
    if (lastUpdate) {
        lastUpdate.textContent = new Date().toLocaleTimeString('ru');
    }
}

function displayErrorMessage(message) {
    console.error(message);
    // Можно добавить отображение уведомлений пользователю
}

function openJaegerTrace(traceId) {
    // Настроить URL Jaeger в зависимости от конфигурации
    const jaegerUrl = `http://localhost:16686/trace/${traceId}`;
    window.open(jaegerUrl, '_blank');
}

// Экспорт функций для глобального использования
window.refreshAllData = refreshAllData;
window.updateStats = updateStats;
window.refreshErrorData = refreshErrorData;
window.openJaegerTrace = openJaegerTrace;
