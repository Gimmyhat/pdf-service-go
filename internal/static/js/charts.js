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

// Format date with local timezone
function formatDate(dateStr) {
    if (!dateStr) return 'N/A';
    const date = new Date(dateStr);
    return date.toLocaleString(undefined, { 
        hour: '2-digit',
        minute: '2-digit',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour12: false
    });
}

// Update statistics
async function updateStats() {
    try {
        const period = document.getElementById('periodSelect').value;
        
        // Добавляем отладочный вывод
        console.log('\n=== Client-side Statistics Update ===');
        console.log('Selected period:', period);
        
        // Всегда добавляем параметр period в URL
        const url = `/api/v1/statistics?period=${encodeURIComponent(period)}`;
        console.log('Requesting URL:', url);

        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();
        
        // Добавляем отладочный вывод полученных данных
        console.log('Received data:', {
            requests: {
                total: data.requests.total,
                success: data.requests.success,
                failed: data.requests.failed,
                byDayOfWeek: data.requests.by_day_of_week,
                byHourOfDay: data.requests.by_hour_of_day
            },
            docx: {
                total: data.docx.total_generations,
                errors: data.docx.error_generations
            },
            gotenberg: {
                total: data.gotenberg.total_requests,
                errors: data.gotenberg.error_requests
            },
            pdf: {
                total: data.pdf.total_files,
                avgSize: data.pdf.average_size,
                minSize: data.pdf.min_size,
                maxSize: data.pdf.max_size
            }
        });

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

        // Update charts with debug output
        console.log('\n=== Updating Charts ===');
        
        if (weekdayChart) {
            console.log('Updating weekday chart with data:', data.requests.by_day_of_week);
            updateWeekdayChart(data.requests.by_day_of_week || {});
        }
        if (hourChart) {
            console.log('Updating hour chart with data:', data.requests.by_hour_of_day);
            updateHourChart(data.requests.by_hour_of_day || {});
        }
        if (docxChart) {
            const docxSuccess = (data.docx.total_generations || 0) - (data.docx.error_generations || 0);
            const docxErrors = data.docx.error_generations || 0;
            console.log('Updating DOCX chart with data:', { success: docxSuccess, errors: docxErrors });
            updateDocxChart(docxSuccess, docxErrors);
        }
        if (gotenbergChart) {
            const gotenbergSuccess = (data.gotenberg.total_requests || 0) - (data.gotenberg.error_requests || 0);
            const gotenbergErrors = data.gotenberg.error_requests || 0;
            console.log('Updating Gotenberg chart with data:', { success: gotenbergSuccess, errors: gotenbergErrors });
            updateGotenbergChart(gotenbergSuccess, gotenbergErrors);
        }
        if (pdfSizeChart) {
            const sizes = [
                parseSize(data.pdf.min_size || '0 B'),
                parseSize(data.pdf.average_size || '0 B'),
                parseSize(data.pdf.max_size || '0 B')
            ];
            console.log('Updating PDF size chart with data:', sizes);
            updatePdfSizeChart(sizes);
        }

        console.log('=== End of Statistics Update ===\n');

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
    
    // Получаем смещение часового пояса в днях (если переход через сутки меняет день недели)
    const timezoneOffset = new Date().getTimezoneOffset() / (60 * 24);
    const days = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'];
    const values = days.map((day, index) => {
        // Вычисляем индекс дня в UTC
        let utcDayIndex = (index + Math.round(timezoneOffset) + 7) % 7;
        const englishDay = Object.keys(dayMapping)[utcDayIndex];
        return data[englishDay] || 0;
    });
    
    weekdayChart.data.datasets[0].data = values;
    weekdayChart.update();
}

// Update hour chart
function updateHourChart(data) {
    const hourData = Array(24).fill(0);
    Object.entries(data).forEach(([hour, count]) => {
        // Преобразуем UTC час в локальное время
        const utcHour = parseInt(hour);
        if (!isNaN(utcHour) && utcHour >= 0 && utcHour < 24) {
            // Получаем смещение в минутах и преобразуем в часы
            const timezoneOffset = new Date().getTimezoneOffset() / 60;
            // Вычисляем локальный час (учитываем переход через сутки)
            let localHour = (utcHour - timezoneOffset + 24) % 24;
            hourData[localHour] = count;
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