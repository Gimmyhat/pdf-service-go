// Chart.js configuration
Chart.defaults.responsive = true;
Chart.defaults.plugins.legend.position = 'bottom';
Chart.defaults.plugins.legend.labels.boxWidth = 12;
Chart.defaults.plugins.legend.labels.padding = 8;

// Color schemes
const colors = {
    primary: '#007bff',
    success: '#28a745',
    danger: '#dc3545',
    warning: '#ffc107',
    info: '#17a2b8'
};

const gradients = {
    primary: createGradient(colors.primary),
    success: createGradient(colors.success),
    danger: createGradient(colors.danger),
    warning: createGradient(colors.warning),
    info: createGradient(colors.info)
};

// Create gradient
function createGradient(color) {
    const ctx = document.createElement('canvas').getContext('2d');
    const gradient = ctx.createLinearGradient(0, 0, 0, 400);
    gradient.addColorStop(0, color);
    gradient.addColorStop(1, '#ffffff');
    return gradient;
}

// Charts instances
let weekdayChart, hourChart, docxChart, pdfSizeChart, gotenbergChart;

// Initialize charts
function initCharts() {
    // Weekday chart
    weekdayChart = new Chart(document.getElementById('weekdayChart'), {
        type: 'bar',
        data: {
            labels: ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'],
            datasets: [{
                label: 'Запросы',
                data: Array(7).fill(0),
                backgroundColor: gradients.primary
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            },
            maintainAspectRatio: false
        }
    });

    // Hour chart
    hourChart = new Chart(document.getElementById('hourChart'), {
        type: 'line',
        data: {
            labels: Array.from({length: 24}, (_, i) => i.toString().padStart(2, '0')),
            datasets: [{
                label: 'Запросы',
                data: Array(24).fill(0),
                borderColor: colors.info,
                backgroundColor: gradients.info,
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            },
            maintainAspectRatio: false
        }
    });

    // DOCX chart
    docxChart = new Chart(document.getElementById('docxChart'), {
        type: 'doughnut',
        data: {
            labels: ['Успешно', 'Ошибки'],
            datasets: [{
                data: [0, 0],
                backgroundColor: [colors.success, colors.danger]
            }]
        },
        options: {
            plugins: {
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const value = context.raw;
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = ((value / total) * 100).toFixed(1);
                            return `${value} (${percentage}%)`;
                        }
                    }
                }
            },
            maintainAspectRatio: false
        }
    });

    // Gotenberg chart
    gotenbergChart = new Chart(document.getElementById('gotenbergChart'), {
        type: 'doughnut',
        data: {
            labels: ['Успешно', 'Ошибки'],
            datasets: [{
                data: [0, 0],
                backgroundColor: [colors.success, colors.danger]
            }]
        },
        options: {
            plugins: {
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            const value = context.raw;
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = ((value / total) * 100).toFixed(1);
                            return `${value} (${percentage}%)`;
                        }
                    }
                }
            },
            maintainAspectRatio: false
        }
    });

    // PDF Size chart
    pdfSizeChart = new Chart(document.getElementById('pdfSizeChart'), {
        type: 'bar',
        data: {
            labels: ['Мин', 'Средний', 'Макс'],
            datasets: [{
                label: 'Размер',
                data: [0, 0, 0],
                backgroundColor: [colors.success, colors.primary, colors.warning]
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            },
            maintainAspectRatio: false
        }
    });
}

// Format date with timezone
function formatDate(dateStr) {
    if (!dateStr) return 'N/A';
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU', { 
        timeZone: 'Europe/Moscow',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Update statistics
async function updateStats() {
    try {
        const period = document.getElementById('periodSelect').value;
        let url = '/api/v1/statistics';
        
        if (period !== 'all') {
            url += `?period=${period}`;
        }

        const response = await fetch(url);
        const data = await response.json();

        // Update summary cards with compact layout
        const summaryData = {
            'Всего запросов': data.requests.total || '0',
            'Успешных': data.requests.success || '0',
            'Ошибок': data.requests.failed || '0',
            'Среднее время': data.requests.average_duration || '0s',
            'DOCX (всего/ошибок)': `${data.docx.total_generations || '0'}/${data.docx.error_generations || '0'}`,
            'Gotenberg (всего/ошибок)': `${data.gotenberg.total_requests || '0'}/${data.gotenberg.error_requests || '0'}`,
            'PDF (всего/средний размер)': `${data.pdf.total_files || '0'}/${data.pdf.average_size || '0 B'}`
        };

        // Обновляем все текстовые данные в компактном виде
        const statsContainer = document.getElementById('statsContainer');
        statsContainer.innerHTML = '';
        for (const [label, value] of Object.entries(summaryData)) {
            statsContainer.innerHTML += `<div class="stat-item"><span class="stat-label">${label}:</span> <span class="stat-value">${value}</span></div>`;
        }

        // Добавляем только одну временную метку (последнее обновление)
        const lastUpdate = Math.max(
            new Date(data.docx.last_generation_time || 0),
            new Date(data.gotenberg.last_request_time || 0),
            new Date(data.pdf.last_processed_time || 0)
        );
        if (lastUpdate > 0) {
            statsContainer.innerHTML += `<div class="stat-item"><span class="stat-label">Последнее обновление:</span> <span class="stat-value">${formatDate(lastUpdate)}</span></div>`;
        }

        // Update charts
        if (weekdayChart) {
            updateWeekdayChart(data.requests.by_day_of_week || {});
        }
        if (hourChart) {
            updateHourChart(data.requests.by_hour_of_day || {});
        }
        if (docxChart) {
            updateDocxChart(
                (data.docx.total_generations || 0) - (data.docx.error_generations || 0),
                data.docx.error_generations || 0
            );
        }
        if (gotenbergChart) {
            updateGotenbergChart(
                (data.gotenberg.total_requests || 0) - (data.gotenberg.error_requests || 0),
                data.gotenberg.error_requests || 0
            );
        }
        if (pdfSizeChart) {
            updatePdfSizeChart([
                parseSize(data.pdf.min_size || '0 B'),
                parseSize(data.pdf.average_size || '0 B'),
                parseSize(data.pdf.max_size || '0 B')
            ]);
        }

    } catch (error) {
        console.error('Failed to fetch statistics:', error);
    }
}

// Parse size string to bytes
function parseSize(sizeStr) {
    const units = {
        'B': 1,
        'KB': 1024,
        'MB': 1024 * 1024,
        'GB': 1024 * 1024 * 1024
    };
    const matches = sizeStr.match(/^([\d.]+)\s*([A-Z]+)$/);
    if (matches) {
        const value = parseFloat(matches[1]);
        const unit = matches[2];
        return value * (units[unit] || 1);
    }
    return 0;
}

// Update weekday chart
function updateWeekdayChart(data) {
    const dayMapping = {
        'Sunday': 'Вс',
        'Monday': 'Пн',
        'Tuesday': 'Вт',
        'Wednesday': 'Ср',
        'Thursday': 'Чт',
        'Friday': 'Пт',
        'Saturday': 'Сб'
    };
    
    const days = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];
    const values = days.map(day => {
        const englishDay = Object.keys(dayMapping).find(key => dayMapping[key] === day);
        return data[englishDay] || 0;
    });
    
    weekdayChart.data.datasets[0].data = values;
    weekdayChart.update();
}

// Update hour chart
function updateHourChart(data) {
    const hourData = Array(24).fill(0);
    Object.entries(data).forEach(([hour, count]) => {
        const hourNumber = parseInt(hour);
        if (!isNaN(hourNumber) && hourNumber >= 0 && hourNumber < 24) {
            hourData[hourNumber] = count;
        }
    });
    hourChart.data.datasets[0].data = hourData;
    hourChart.update();
}

// Update DOCX chart
function updateDocxChart(success, failed) {
    docxChart.data.datasets[0].data = [success, failed];
    docxChart.update();
}

// Update Gotenberg chart
function updateGotenbergChart(success, failed) {
    gotenbergChart.data.datasets[0].data = [success, failed];
    gotenbergChart.update();
}

// Update PDF size chart
function updatePdfSizeChart(data) {
    pdfSizeChart.data.datasets[0].data = data;
    pdfSizeChart.update();
}

// Initialize charts and start updating
document.addEventListener('DOMContentLoaded', () => {
    initCharts();
    updateStats();
    // Update every 30 seconds
    setInterval(updateStats, 30000);
}); 