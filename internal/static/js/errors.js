// Error Analysis Dashboard JavaScript

let currentData = null;

// Загрузка данных при запуске страницы
document.addEventListener('DOMContentLoaded', function() {
    refreshData();
    // Автообновление каждые 30 секунд
    setInterval(refreshData, 30000);
});

// Обновление данных
async function refreshData() {
    try {
        const period = document.getElementById('periodFilter').value;
        const type = document.getElementById('typeFilter').value;
        const component = document.getElementById('componentFilter').value;
        const severity = document.getElementById('severityFilter').value;

        // Построение URL с параметрами
        const params = new URLSearchParams({
            period: period,
            limit: 50
        });
        
        if (type) params.append('type', type);
        if (component) params.append('component', component);
        if (severity) params.append('severity', severity);

        // Получение данных об ошибках
        const errorsResponse = await fetch(`/api/v1/errors?${params}`);
        const errorsData = await errorsResponse.json();

        // Получение статистики здоровья
        const statsResponse = await fetch(`/api/v1/errors/stats?period=${period}`);
        const statsData = await statsResponse.json();

        currentData = { errors: errorsData, stats: statsData };
        
        updateHealthStatus(statsData);
        updateErrorSummary(statsData);
        updateRecentErrors(errorsData.summary.recent_errors);
        
    } catch (error) {
        console.error('Error fetching data:', error);
        showError('Failed to load error data');
    }
}

// Обновление статуса здоровья
function updateHealthStatus(data) {
    const healthElement = document.getElementById('healthStatus');
    const status = data.health_status || 'unknown';
    
    let indicator = 'health-healthy';
    let text = 'System Healthy';
    
    switch (status) {
        case 'critical':
            indicator = 'health-critical';
            text = `Critical - ${data.errors_1h} errors in last hour`;
            break;
        case 'warning':
            indicator = 'health-warning';
            text = `Warning - ${data.errors_1h} errors in last hour`;
            break;
        case 'degraded':
            indicator = 'health-degraded';
            text = `Degraded - ${data.errors_24h} errors in 24h`;
            break;
        default:
            text = `Healthy - ${data.errors_1h || 0} errors in last hour`;
    }
    
    healthElement.innerHTML = `
        <span class="health-indicator ${indicator}"></span>
        <span>${text}</span>
    `;
}

// Обновление сводки ошибок
function updateErrorSummary(data) {
    document.getElementById('totalErrors').textContent = data.total_errors || 0;
    document.getElementById('errors24h').textContent = data.errors_24h || 0;
    document.getElementById('errors1h').textContent = data.errors_1h || 0;
}

// Обновление списка последних ошибок
function updateRecentErrors(errors) {
    const container = document.getElementById('recentErrors');
    
    if (!errors || errors.length === 0) {
        container.innerHTML = `
            <div class="text-center text-muted">
                <i class="bi bi-check-circle-fill text-success"></i>
                <p class="mt-2">No errors found for the selected period</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = errors.map(error => createErrorCard(error)).join('');
}

// Создание карточки ошибки
function createErrorCard(error) {
    const timestamp = new Date(error.timestamp).toLocaleString();
    const duration = error.duration_ms ? `${error.duration_ms}ms` : 'N/A';
    const solutions = error.request_details?.solutions || [];
    
    return `
        <div class="card error-card ${error.severity}">
            <div class="card-header d-flex justify-content-between align-items-center">
                <div>
                    <span class="badge bg-danger error-badge">${error.error_type}</span>
                    <span class="badge bg-secondary error-badge">${error.component}</span>
                    <span class="badge bg-${getSeverityColor(error.severity)} error-badge">${error.severity}</span>
                    ${error.http_status ? `<span class="badge bg-dark error-badge">HTTP ${error.http_status}</span>` : ''}
                </div>
                <small class="text-muted">${timestamp}</small>
            </div>
            <div class="card-body">
                <div class="row">
                    <div class="col-md-8">
                        <h6 class="card-title">Error Message</h6>
                        <p class="card-text">${escapeHtml(error.message)}</p>
                        
                        ${error.request_id ? `<p><strong>Request ID:</strong> <code>${error.request_id}</code></p>` : ''}
                        ${error.client_ip ? `<p><strong>Client IP:</strong> ${error.client_ip}</p>` : ''}
                        ${error.http_method && error.http_path ? `<p><strong>Request:</strong> ${error.http_method} ${error.http_path}</p>` : ''}
                        <p><strong>Duration:</strong> ${duration}</p>
                        
                        ${error.trace_id ? `
                            <p><strong>Trace:</strong> 
                                <a href="http://localhost:16686/trace/${error.trace_id}" target="_blank" class="jaeger-link">
                                    <i class="bi bi-box-arrow-up-right"></i> View in Jaeger
                                </a>
                            </p>
                        ` : ''}
                        
                        ${error.stack_trace ? `
                            <div class="mt-3">
                                <h6>Stack Trace</h6>
                                <div class="stack-trace">${escapeHtml(error.stack_trace)}</div>
                            </div>
                        ` : ''}
                    </div>
                    <div class="col-md-4">
                        ${solutions.length > 0 ? `
                            <div class="solution-list">
                                <h6><i class="bi bi-lightbulb"></i> Suggested Solutions</h6>
                                <ul>
                                    ${solutions.map(solution => `<li>${escapeHtml(solution)}</li>`).join('')}
                                </ul>
                            </div>
                        ` : ''}
                        
                        ${error.request_details ? `
                            <div class="mt-3">
                                <h6>Request Details</h6>
                                <small class="text-muted">
                                    ${Object.entries(error.request_details)
                                        .filter(([key]) => key !== 'solutions')
                                        .map(([key, value]) => `<strong>${key}:</strong> ${value}`)
                                        .join('<br>')
                                    }
                                </small>
                            </div>
                        ` : ''}
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Получение цвета для уровня серьезности
function getSeverityColor(severity) {
    switch (severity) {
        case 'critical': return 'danger';
        case 'high': return 'warning';
        case 'medium': return 'info';
        case 'low': return 'secondary';
        default: return 'secondary';
    }
}

// Экранирование HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Показ ошибки
function showError(message) {
    const container = document.getElementById('recentErrors');
    container.innerHTML = `
        <div class="alert alert-danger" role="alert">
            <i class="bi bi-exclamation-triangle"></i>
            ${message}
        </div>
    `;
}

// Экспорт данных
function exportData() {
    if (!currentData) {
        alert('No data to export');
        return;
    }
    
    const dataStr = JSON.stringify(currentData, null, 2);
    const dataBlob = new Blob([dataStr], {type: 'application/json'});
    const url = URL.createObjectURL(dataBlob);
    
    const link = document.createElement('a');
    link.href = url;
    link.download = `error-analysis-${new Date().toISOString().split('T')[0]}.json`;
    link.click();
    
    URL.revokeObjectURL(url);
}