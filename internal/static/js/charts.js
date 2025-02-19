// Chart.js configuration
Chart.defaults.responsive = true;
Chart.defaults.plugins.legend.position = 'bottom';

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
            labels: ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
            datasets: [{
                label: 'Requests',
                data: Array(7).fill(0),
                backgroundColor: gradients.primary
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            }
        }
    });

    // Hour chart
    hourChart = new Chart(document.getElementById('hourChart'), {
        type: 'line',
        data: {
            labels: Array.from({length: 24}, (_, i) => i.toString().padStart(2, '0') + ':00'),
            datasets: [{
                label: 'Requests',
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
            }
        }
    });

    // DOCX chart
    docxChart = new Chart(document.getElementById('docxChart'), {
        type: 'doughnut',
        data: {
            labels: ['Success', 'Failed'],
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
            }
        }
    });

    // Gotenberg chart
    gotenbergChart = new Chart(document.getElementById('gotenbergChart'), {
        type: 'doughnut',
        data: {
            labels: ['Success', 'Failed'],
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
            }
        }
    });

    // PDF Size chart
    pdfSizeChart = new Chart(document.getElementById('pdfSizeChart'), {
        type: 'bar',
        data: {
            labels: ['Min', 'Average', 'Max'],
            datasets: [{
                label: 'File Size',
                data: [0, 0, 0],
                backgroundColor: [colors.success, colors.primary, colors.warning]
            }]
        },
        options: {
            scales: {
                y: {
                    beginAtZero: true
                }
            }
        }
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

        // Update summary cards
        document.getElementById('totalRequests').textContent = data.requests.total || '0';
        document.getElementById('successfulRequests').textContent = data.requests.success || '0';
        document.getElementById('failedRequests').textContent = data.requests.failed || '0';
        document.getElementById('avgDuration').textContent = data.requests.average_duration || '0s';

        // Update DOCX stats
        document.getElementById('docxTotal').textContent = data.docx.total_generations || '0';
        document.getElementById('docxErrors').textContent = data.docx.error_generations || '0';
        document.getElementById('docxAvgDuration').textContent = data.docx.average_duration || '0s';
        document.getElementById('docxLastGeneration').textContent = data.docx.last_generation_time ? new Date(data.docx.last_generation_time).toLocaleString() : 'N/A';

        // Update Gotenberg stats
        document.getElementById('gotenbergTotal').textContent = data.gotenberg.total_requests || '0';
        document.getElementById('gotenbergErrors').textContent = data.gotenberg.error_requests || '0';
        document.getElementById('gotenbergAvgDuration').textContent = data.gotenberg.average_duration || '0s';
        document.getElementById('gotenbergLastRequest').textContent = data.gotenberg.last_request_time ? new Date(data.gotenberg.last_request_time).toLocaleString() : 'N/A';

        // Update PDF stats
        document.getElementById('pdfTotal').textContent = data.pdf.total_files || '0';
        document.getElementById('pdfAvgSize').textContent = data.pdf.average_size || '0 B';
        document.getElementById('pdfLastProcessed').textContent = data.pdf.last_processed_time ? new Date(data.pdf.last_processed_time).toLocaleString() : 'N/A';

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
    weekdayChart.data.datasets[0].data = Object.values(data);
    weekdayChart.update();
}

// Update hour chart
function updateHourChart(data) {
    hourChart.data.datasets[0].data = Object.values(data);
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