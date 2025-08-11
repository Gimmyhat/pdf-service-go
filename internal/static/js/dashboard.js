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
                case '#archive':
                    // Подгружаем архив при первом открытии вкладки
                    initArchiveInfiniteScroll();
                    refreshArchive();
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
        case '#archive':
            refreshArchive();
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
        // Загружаем статистику ошибок (старая система)
        const statsResponse = await fetch(`/api/v1/errors/stats?period=${period}`);
        const stats = await statsResponse.json();
        
        // Загружаем детальную информацию об ошибках (старая система)
        let errorsUrl = `/api/v1/errors?period=${period}&limit=25`;
        if (type) errorsUrl += `&type=${type}`;
        if (component) errorsUrl += `&component=${component}`;
        if (severity) errorsUrl += `&severity=${severity}`;
        
        // Загружаем детальные запросы с ошибками (новая система)
        let requestErrorsUrl = `/api/v1/requests/error?period=${period}&limit=25`;
        if (component) requestErrorsUrl += `&category=${component}_error`;
        
        const [errorsResponse, requestErrorsResponse] = await Promise.all([
            fetch(errorsUrl),
            fetch(requestErrorsUrl)
        ]);
        
        const errors = await errorsResponse.json();
        const requestErrors = await requestErrorsResponse.json();
        
        console.log('Loaded error data:', { stats, errors, requestErrors });
        
        errorData = { stats, errors, requestErrors };
        displayErrorStats(stats, requestErrors);
        
        const oldErrors = errors.summary?.recent_errors || [];
        const newErrors = requestErrors.error_requests || [];
        
        console.log('Old errors:', oldErrors.length, 'New errors:', newErrors.length);
        
        displayCombinedErrorList(oldErrors, newErrors);
        
    } catch (error) {
        console.error('Error loading error data:', error);
        displayErrorMessage('Ошибка загрузки данных об ошибках');
    }
}

function displayErrorStats(stats, requestErrors) {
    // Комбинируем статистику из обеих систем
    const totalErrors = (stats.total_errors || 0) + (requestErrors?.total_found || 0);
    const errors24h = (stats.errors_24h || 0) + (requestErrors?.analytics?.by_category ? Object.values(requestErrors.analytics.by_category).reduce((a, b) => a + b, 0) : 0);
    const errors1h = stats.errors_1h || 0; // Для часовой статистики используем только старую систему
    
    document.getElementById('errorTotalErrors').textContent = formatNumber(totalErrors);
    document.getElementById('errorErrors24h').textContent = formatNumber(errors24h);
    document.getElementById('errorErrors1h').textContent = formatNumber(errors1h);
}

function displayCombinedErrorList(oldErrors, requestErrors) {
    const container = document.getElementById('errorRecentErrors');
    if (!container) {
        console.error('Container errorRecentErrors not found');
        return;
    }
    
    console.log('displayCombinedErrorList called with:', { 
        oldErrors: oldErrors?.length || 0, 
        requestErrors: requestErrors?.length || 0 
    });
    
    const allErrors = [...(oldErrors || []), ...(requestErrors || [])];
    
    console.log('Total errors to display:', allErrors.length);
    console.log('Sample errors:', allErrors.slice(0, 2));
    
    if (allErrors.length === 0) {
        container.innerHTML = '<div class="alert alert-success">Нет ошибок за выбранный период</div>';
        return;
    }
    
    // Сортируем все ошибки по времени (новые сначала)
    allErrors.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
    
    let html = '';
    let requestErrorCount = 0;
    let oldErrorCount = 0;
    
    allErrors.forEach((error, index) => {
        try {
            // Определяем тип ошибки (старая система или новая)
            const isRequestError = error.body_text !== undefined;
            
            console.log(`Error ${index}:`, { isRequestError, error_id: error.id || error.request_id, type: error.error_category || error.error_type });
            
            if (isRequestError) {
                requestErrorCount++;
                html += generateRequestErrorCard(error);
            } else {
                oldErrorCount++;
                html += generateOldErrorCard(error);
            }
        } catch (e) {
            console.error('Error generating card for:', error, e);
        }
    });
    
    console.log(`Generated ${requestErrorCount} request cards and ${oldErrorCount} old cards`);
    
    container.innerHTML = html;
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
        // Новые категории ошибок запросов
        case 'client_error': return 'warning';
        case 'server_error': return 'danger';
        case 'validation_error': return 'info';
        case 'timeout_error': return 'danger';
        case 'not_found': return 'secondary';
        case 'instant_failure': return 'danger';
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
// Функции для генерации карточек ошибок
function generateRequestErrorCard(error) {
    const severityClass = getSeverityClass(error.error_category);
    const timeAgo = new Date(error.timestamp).toLocaleString('ru');
    const duration = error.duration_ns ? (error.duration_ns / 1000000).toFixed(2) + 'ms' : 'N/A';
    
    return `
        <div class="error-card card ${error.error_category}" style="margin-bottom: 1rem;">
            <div class="card-body">
                <div class="d-flex justify-content-between align-items-start mb-2">
                    <h6 class="card-title mb-0">
                        <span class="badge bg-${severityClass} error-badge">REQUEST</span>
                        <span class="badge bg-info">NEW</span>
                        ${error.path || 'Unknown Path'}
                    </h6>
                    <small class="text-muted">${timeAgo}</small>
                </div>
                
                <p class="card-text">${error.method} ${error.path} - ${error.error_category}</p>
                
                <div class="row text-small mb-2">
                    <div class="col-sm-6">
                        <strong>HTTP Status:</strong> ${error.http_status || 'N/A'}
                    </div>
                    <div class="col-sm-6">
                        <strong>Duration:</strong> ${duration}
                    </div>
                    <div class="col-sm-6">
                        <strong>Client IP:</strong> ${error.client_ip || 'N/A'}
                    </div>
                    <div class="col-sm-6">
                        <strong>Request ID:</strong> ${error.request_id || 'N/A'}
                    </div>
                </div>
                
                <div class="mt-2">
                    <button class="btn btn-sm btn-outline-primary" type="button" 
                            onclick="showRequestBodyFromApi('${error.request_id}')">
                        <i class="bi bi-file-text"></i> Показать тело запроса
                    </button>
                    ${error.body_size_bytes ? `<small class="text-muted ms-2">(${error.body_size_bytes} bytes)</small>` : ''}
                </div>
                
                ${error.headers ? `
                    <div class="mt-2">
                        <button class="btn btn-sm btn-outline-secondary" type="button" 
                                data-bs-toggle="collapse" data-bs-target="#headers-${error.id}" 
                                aria-expanded="false">
                            <i class="bi bi-list"></i> Показать заголовки
                        </button>
                        <div class="collapse mt-2" id="headers-${error.id}">
                            <div class="stack-trace">
                                ${Object.entries(error.headers).map(([key, value]) => `${key}: ${value}`).join('\\n')}
                            </div>
                        </div>
                    </div>
                ` : ''}
            </div>
        </div>
    `;
}

function generateOldErrorCard(error) {
    const severityClass = getSeverityClass(error.severity);
    const timeAgo = new Date(error.timestamp).toLocaleString('ru');
    
    return `
        <div class="error-card card ${error.severity}" style="margin-bottom: 1rem;">
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
}

// Функция для показа тела запроса из API
async function showRequestBodyFromApi(requestId) {
    try {
        console.log('Fetching request body for:', requestId);
        
        // Сначала получаем детали запроса
        const detailResponse = await fetch(`/api/v1/requests/${requestId}`);
        if (!detailResponse.ok) {
            throw new Error(`HTTP ${detailResponse.status}: ${detailResponse.statusText}`);
        }
        
        const requestDetailRaw = await detailResponse.json();
        const requestDetail = requestDetailRaw.request_detail || requestDetailRaw;
        console.log('Request detail:', requestDetail);
        
        // Затем получаем тело запроса
        const bodyResponse = await fetch(`/api/v1/requests/${requestId}/body`);
        if (!bodyResponse.ok) {
            throw new Error(`HTTP ${bodyResponse.status}: ${bodyResponse.statusText}`);
        }
        
        const bodyData = await bodyResponse.json();
        console.log('Body data:', bodyData);
        
        const bodyText = bodyData.body || bodyData.body_text || 'Тело запроса пустое';
        
        showRequestBody(requestId, bodyText, requestDetail);
        
    } catch (error) {
        console.error('Error fetching request body:', error);
        alert(`Ошибка загрузки тела запроса: ${error.message}`);
    }
}

// Функция для показа тела запроса в модальном окне
function showRequestBody(requestId, bodyText, requestDetail = null) {
    const detailsSection = requestDetail ? `
        <div class="row mb-3">
            <div class="col-md-6">
                <strong>Метод:</strong> ${requestDetail.method ?? '—'}<br>
                <strong>Путь:</strong> ${requestDetail.path ?? '—'}<br>
                <strong>HTTP Статус:</strong> ${requestDetail.http_status ?? '—'}
            </div>
            <div class="col-md-6">
                <strong>Client IP:</strong> ${requestDetail.client_ip ?? '—'}<br>
                <strong>Длительность:</strong> ${requestDetail.duration_ns ? (requestDetail.duration_ns / 1000000).toFixed(2) + 'ms' : 'N/A'}<br>
                <strong>Размер:</strong> ${requestDetail.body_size_bytes ?? 0} bytes
            </div>
        </div>
        <hr>
    ` : '';

    const modalHtml = `
        <div class="modal fade" id="requestBodyModal" tabindex="-1" aria-labelledby="requestBodyModalLabel" aria-hidden="true">
            <div class="modal-dialog modal-xl">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="requestBodyModalLabel">
                            <i class="bi bi-file-text"></i> Детали запроса
                        </h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div class="mb-3">
                            <label class="form-label"><strong>Request ID:</strong></label>
                            <code>${requestId}</code>
                        </div>
                        
                        ${detailsSection}
                        
                        <div class="mb-3">
                            <label class="form-label"><strong>Тело запроса:</strong></label>
                            <pre class="bg-light p-3 border rounded" style="max-height: 400px; overflow-y: auto;"><code id="request-body-content">${bodyText}</code></pre>
                        </div>
                        
                        <div class="d-flex gap-2">
                            <button class="btn btn-sm btn-outline-primary" onclick="copyToClipboard(\`${bodyText.replace(/`/g, '\\`').replace(/\$/g, '\\$')}\`)">
                                <i class="bi bi-clipboard"></i> Копировать
                            </button>
                            <button class="btn btn-sm btn-outline-secondary" onclick="formatJSON()">
                                <i class="bi bi-code"></i> Форматировать JSON
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Удаляем существующий модал если есть
    const existingModal = document.getElementById('requestBodyModal');
    if (existingModal) {
        existingModal.remove();
    }
    
    // Добавляем новый модал
    document.body.insertAdjacentHTML('beforeend', modalHtml);
    
    // Показываем модал
    const modal = new bootstrap.Modal(document.getElementById('requestBodyModal'));
    modal.show();
}

// Утилиты
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        alert('Скопировано в буфер обмена!');
    }).catch(err => {
        console.error('Error copying to clipboard:', err);
    });
}

function formatJSON() {
    try {
        const codeElement = document.querySelector('#request-body-content');
        const text = codeElement.textContent;
        const formatted = JSON.stringify(JSON.parse(text), null, 2);
        codeElement.textContent = formatted;
    } catch (e) {
        alert('Не удалось отформатировать как JSON: ' + e.message);
    }
}

window.refreshErrorData = refreshErrorData;
window.openJaegerTrace = openJaegerTrace;
window.showRequestBody = showRequestBody;
window.showRequestBodyFromApi = showRequestBodyFromApi;

// ===== Архив запросов =====
let archiveData = {
    items: [],
    totalCount: 0,
    offset: 0,
    limit: 25,
    hasMore: false
};

async function refreshArchive(append = false) {
    try {
        if (!append) {
            archiveData.offset = 0;
            archiveData.items = [];
        }
        
        const limit = document.getElementById('archiveLimit')?.value || '25';
        archiveData.limit = parseInt(limit);
        
        const resp = await fetch(`/api/v1/requests/recent?limit=${archiveData.limit}&offset=${archiveData.offset}`);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        const data = await resp.json();
        
        if (append) {
            archiveData.items.push(...(data.recent_requests || []));
        } else {
            archiveData.items = data.recent_requests || [];
        }
        
        archiveData.totalCount = data.total_count || 0;
        archiveData.hasMore = data.has_more || false;
        archiveData.offset = data.offset + data.recent_requests.length;
        
        renderArchiveTable(archiveData.items);
        updateArchivePagination();
        
    } catch (e) {
        console.error('Error loading archive:', e);
        const tbody = document.querySelector('#archiveTable tbody');
        if (tbody) tbody.innerHTML = `<tr><td colspan="9" class="text-center text-danger">Ошибка загрузки архива</td></tr>`;
    }
}

function updateArchivePagination() {
    let paginationHtml = '';
    if (archiveData.hasMore) {
        paginationHtml = `
            <div class="text-center mt-3">
                <button class="btn btn-outline-primary" onclick="loadMoreArchive()">
                    <i class="bi bi-arrow-down"></i> Загрузить ещё (показано ${archiveData.items.length} из ${archiveData.totalCount})
                </button>
            </div>
        `;
    } else if (archiveData.totalCount > 0) {
        paginationHtml = `
            <div class="text-center mt-3">
                <small class="text-muted">Показано ${archiveData.items.length} из ${archiveData.totalCount} записей</small>
            </div>
        `;
    }
    
    const container = document.querySelector('#archiveTable').parentElement.parentElement;
    let paginationDiv = container.querySelector('.archive-pagination');
    if (!paginationDiv) {
        paginationDiv = document.createElement('div');
        paginationDiv.className = 'archive-pagination';
        container.appendChild(paginationDiv);
    }
    paginationDiv.innerHTML = paginationHtml;
}

async function loadMoreArchive() {
    await refreshArchive(true);
}

// Инициализация бесконечной прокрутки для архива
let archiveObserverInitialized = false;
function initArchiveInfiniteScroll() {
    if (archiveObserverInitialized) return;
    const sentinel = document.getElementById('archiveSentinel');
    if (!sentinel) return;
    const observer = new IntersectionObserver(async (entries) => {
        for (const entry of entries) {
            if (entry.isIntersecting) {
                if (archiveData.hasMore) {
                    await refreshArchive(true);
                }
            }
        }
    }, { root: null, rootMargin: '0px', threshold: 1.0 });
    observer.observe(sentinel);
    archiveObserverInitialized = true;
}

function renderArchiveTable(items) {
    const tbody = document.querySelector('#archiveTable tbody');
    if (!tbody) return;
    if (!items || items.length === 0) {
        tbody.innerHTML = `<tr><td colspan="9" class="text-center text-muted">Нет данных</td></tr>`;
        return;
    }
    const rows = items.map(it => {
        const ts = it.timestamp ? new Date(it.timestamp).toLocaleString('ru') : '—';
        const dur = typeof it.duration_ns === 'number' ? (it.duration_ns / 1e6).toFixed(1) + 'ms' : '—';
        const bodySize = typeof it.body_size_bytes === 'number' ? formatBytes(it.body_size_bytes) : '—';
        const statusBadge = it.success ? '<span class="badge bg-success">OK</span>' : `<span class="badge bg-danger">${it.http_status ?? 'ERR'}</span>`;
        const reqLink = it.request_file_path ? `<a href="${it.request_file_path}" target="_blank">json</a>` : '—';
        const resLink = it.result_file_path ? `<a href="${it.result_file_path}" target="_blank">pdf</a>${(typeof it.result_size_bytes === 'number') ? ` <small class="text-muted">(${formatBytes(it.result_size_bytes)})</small>` : ''}` : '—';
        const viewBtn = it.request_id ? `<button class="btn btn-sm btn-outline-primary" onclick="showRequestBodyFromApi('${it.request_id}')"><i class="bi bi-eye"></i></button>` : '';
        return `<tr>
            <td>${ts}</td>
            <td>${it.method ?? '—'}</td>
            <td><code>${it.path ?? '—'}</code></td>
            <td>${statusBadge}</td>
            <td>${dur}</td>
            <td>${bodySize}</td>
            <td>${reqLink}</td>
            <td>${resLink}</td>
            <td class="text-end">${viewBtn}</td>
        </tr>`;
    }).join('');
    tbody.innerHTML = rows;
}

async function cleanupArchive() {
    try {
        const keep = 100;
        const resp = await fetch(`/api/v1/requests/cleanup?keep=${keep}`, { method: 'POST' });
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        await refreshArchive();
        alert('Очистка выполнена');
    } catch (e) {
        console.error('Cleanup error:', e);
        alert('Ошибка очистки');
    }
}

window.refreshArchive = refreshArchive;
window.cleanupArchive = cleanupArchive;
