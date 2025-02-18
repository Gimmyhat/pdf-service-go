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
let weekdayChart, hourChart, docxChart, pdfSizeChart;

// Initialize charts
function initCharts() {
    // Weekday chart
    weekdayChart = new Chart(document.getElementById('weekdayChart'), {
        type: 'bar',
        data: {
            labels: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'],
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
        }
    });

    // PDF Size chart
    pdfSizeChart = new Chart(document.getElementById('pdfSizeChart'), {
        type: 'bar',
        data: {
            labels: ['0-1MB', '1-5MB', '5-10MB', '10-20MB', '20MB+'],
            datasets: [{
                label: 'Files',
                data: Array(5).fill(0),
                backgroundColor: gradients.warning
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
        const response = await fetch('/api/v1/statistics');
        const data = await response.json();

        // Update summary cards
        document.getElementById('totalRequests').textContent = data.requests.total;
        document.getElementById('successfulRequests').textContent = data.requests.success;
        document.getElementById('failedRequests').textContent = data.requests.failed;
        document.getElementById('avgDuration').textContent = data.requests.average_duration;

        // Convert day of week data
        const weekdayData = [0, 0, 0, 0, 0, 0, 0];
        Object.entries(data.requests.by_day_of_week).forEach(([day, count]) => {
            const dayIndex = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'].indexOf(day);
            if (dayIndex !== -1) {
                weekdayData[dayIndex] = count;
            }
        });

        // Convert hour data
        const hourData = Array(24).fill(0);
        Object.entries(data.requests.by_hour_of_day).forEach(([hour, count]) => {
            const hourIndex = parseInt(hour.split(':')[0]);
            if (!isNaN(hourIndex) && hourIndex >= 0 && hourIndex < 24) {
                hourData[hourIndex] = count;
            }
        });

        // Update charts
        updateWeekdayChart(weekdayData);
        updateHourChart(hourData);
        updateDocxChart(data.docx.total_generations - data.docx.error_generations, data.docx.error_generations);

        // Convert PDF size data
        const pdfSizeData = [0, 0, 0, 0, 0]; // 0-1MB, 1-5MB, 5-10MB, 10-20MB, 20MB+
        if (data.pdf.total_files > 0) {
            const sizeInMB = parseFloat(data.pdf.average_size.split(' ')[0]) / 1024; // Convert KB to MB
            const sizeIndex = sizeInMB <= 1 ? 0 :
                            sizeInMB <= 5 ? 1 :
                            sizeInMB <= 10 ? 2 :
                            sizeInMB <= 20 ? 3 : 4;
            pdfSizeData[sizeIndex] = data.pdf.total_files;
        }
        updatePdfSizeChart(pdfSizeData);

    } catch (error) {
        console.error('Failed to fetch statistics:', error);
    }
}

// Format duration in milliseconds to human-readable format
function formatDuration(ms) {
    if (ms < 1000) return ms + 'ms';
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return seconds + 's';
    const minutes = Math.floor(seconds / 60);
    return minutes + 'm ' + (seconds % 60) + 's';
}

// Update weekday chart
function updateWeekdayChart(data) {
    weekdayChart.data.datasets[0].data = data;
    weekdayChart.update();
}

// Update hour chart
function updateHourChart(data) {
    hourChart.data.datasets[0].data = data;
    hourChart.update();
}

// Update DOCX chart
function updateDocxChart(success, failed) {
    docxChart.data.datasets[0].data = [success, failed];
    docxChart.update();
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