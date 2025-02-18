// Конфигурация графиков
const chartConfig = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
        legend: {
            position: 'top',
        }
    }
};

// Цвета для графиков
const colors = {
    primary: 'rgb(54, 162, 235)',
    success: 'rgb(75, 192, 192)',
    danger: 'rgb(255, 99, 132)',
    warning: 'rgb(255, 205, 86)'
};

// Создание графиков
let weekdayChart, hourChart, docxChart, pdfChart;

// Функция обновления статистики
async function updateStatistics() {
    try {
        const response = await fetch('/api/v1/statistics');
        const data = await response.json();

        // Обновляем карточки с общей статистикой
        document.getElementById('total-requests').textContent = data.requests.total;
        document.getElementById('success-requests').textContent = data.requests.success;
        document.getElementById('failed-requests').textContent = data.requests.failed;
        document.getElementById('avg-duration').textContent = data.requests.average_duration;

        // Обновляем график по дням недели
        updateWeekdayChart(data.requests.by_day_of_week);

        // Обновляем график по часам
        updateHourChart(data.requests.by_hour_of_day);

        // Обновляем график DOCX
        updateDocxChart(data.docx);

        // Обновляем график PDF
        updatePdfChart(data.pdf);

    } catch (error) {
        console.error('Error fetching statistics:', error);
    }
}

// Функция обновления графика по дням недели
function updateWeekdayChart(data) {
    const days = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    const values = days.map(day => data[day] || 0);

    if (!weekdayChart) {
        const ctx = document.getElementById('weekdayChart').getContext('2d');
        weekdayChart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'],
                datasets: [{
                    label: 'Количество запросов',
                    data: values,
                    backgroundColor: colors.primary
                }]
            },
            options: chartConfig
        });
    } else {
        weekdayChart.data.datasets[0].data = values;
        weekdayChart.update();
    }
}

// Функция обновления графика по часам
function updateHourChart(data) {
    const hours = Array.from({length: 24}, (_, i) => `${i.toString().padStart(2, '0')}:00`);
    const values = hours.map(hour => data[hour] || 0);

    if (!hourChart) {
        const ctx = document.getElementById('hourChart').getContext('2d');
        hourChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: hours,
                datasets: [{
                    label: 'Количество запросов',
                    data: values,
                    borderColor: colors.success,
                    fill: false,
                    tension: 0.4
                }]
            },
            options: chartConfig
        });
    } else {
        hourChart.data.datasets[0].data = values;
        hourChart.update();
    }
}

// Функция обновления графика DOCX
function updateDocxChart(data) {
    if (!docxChart) {
        const ctx = document.getElementById('docxChart').getContext('2d');
        docxChart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Успешно', 'Ошибки'],
                datasets: [{
                    data: [
                        data.total_generations - data.error_generations,
                        data.error_generations
                    ],
                    backgroundColor: [colors.success, colors.danger]
                }]
            },
            options: {
                ...chartConfig,
                plugins: {
                    ...chartConfig.plugins,
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const value = context.raw;
                                const total = data.total_generations;
                                const percentage = ((value / total) * 100).toFixed(1);
                                return `${value} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    } else {
        docxChart.data.datasets[0].data = [
            data.total_generations - data.error_generations,
            data.error_generations
        ];
        docxChart.update();
    }
}

// Функция обновления графика PDF
function updatePdfChart(data) {
    if (!pdfChart) {
        const ctx = document.getElementById('pdfChart').getContext('2d');
        pdfChart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['Минимальный', 'Средний', 'Максимальный'],
                datasets: [{
                    label: 'Размер файла',
                    data: [data.min_size, data.average_size, data.max_size],
                    backgroundColor: [colors.success, colors.primary, colors.warning]
                }]
            },
            options: chartConfig
        });
    } else {
        pdfChart.data.datasets[0].data = [data.min_size, data.average_size, data.max_size];
        pdfChart.update();
    }
}

// Обновляем статистику каждые 30 секунд
updateStatistics();
setInterval(updateStatistics, 30000); 