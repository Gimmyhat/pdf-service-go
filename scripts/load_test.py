#!/usr/bin/env python3
import aiohttp
import asyncio
import json
import time
import argparse
from datetime import datetime
import statistics
import sys
from collections import defaultdict
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

class LoadTester:
    def __init__(self, url, concurrency, total_requests, request_data):
        self.url = url
        self.concurrency = concurrency
        self.total_requests = total_requests
        self.request_data = request_data
        self.results = []
        self.errors = []
        self.start_time = None
        self.end_time = None
        self.response_sizes = []
        # Добавляем словари для хранения метрик по этапам
        self.stage_times = defaultdict(list)
        self.stage_stats = defaultdict(dict)

    async def make_request(self, session, request_id):
        print(f"Starting request {request_id}")
        try:
            # Добавляем заголовки для трейсинга
            headers = {
                'X-Request-ID': f'load-test-{request_id}',
                'Accept-Encoding': 'gzip',
            }
            
            start_time = time.time()
            async with session.post(self.url, json=self.request_data, headers=headers) as response:
                response_time = time.time() - start_time
                status = response.status
                
                if status != 200:
                    error_text = await response.text()
                    error_msg = f"Request {request_id}: Status {status}, Error: {error_text}"
                    print(f"Error on request {request_id}: {error_msg}")
                    self.errors.append(error_msg)
                else:
                    content = await response.read()
                    self.response_sizes.append(len(content))
                    self.results.append(response_time)
                    
                    # Получаем метрики времени из заголовков ответа
                    stage_times = {
                        'docx_generation': float(response.headers.get('X-Docx-Generation-Time', 0)),
                        'pdf_conversion': float(response.headers.get('X-PDF-Conversion-Time', 0)),
                        'total_processing': float(response.headers.get('X-Total-Processing-Time', 0))
                    }
                    
                    # Сохраняем метрики по этапам
                    for stage, time_value in stage_times.items():
                        self.stage_times[stage].append(time_value)
                    
                    print(f"Request {request_id} completed in {response_time:.2f}s, "
                          f"DOCX: {stage_times['docx_generation']:.2f}s, "
                          f"PDF: {stage_times['pdf_conversion']:.2f}s, "
                          f"Total: {stage_times['total_processing']:.2f}s")
                
                return status, response_time
        except Exception as e:
            error_msg = f"Request {request_id}: Exception: {str(e)}"
            print(f"Error on request {request_id}: {error_msg}")
            self.errors.append(error_msg)
            return None, None

    async def run_test(self):
        print("Starting test execution...")
        self.start_time = time.time()
        async with aiohttp.ClientSession() as session:
            tasks = []
            for i in range(self.total_requests):
                print(f"Creating task {i + 1}")
                task = asyncio.create_task(self.make_request(session, i + 1))
                tasks.append(task)
                if len(tasks) >= self.concurrency:
                    print(f"Executing batch of {len(tasks)} tasks")
                    await asyncio.gather(*tasks)
                    tasks = []
                    print(f"Completed batch of {self.concurrency} requests")
            if tasks:
                print(f"Executing final batch of {len(tasks)} tasks")
                await asyncio.gather(*tasks)
        self.end_time = time.time()
        print("Test execution completed")

    def analyze_stages(self):
        """Анализирует статистику по этапам обработки"""
        for stage, times in self.stage_times.items():
            if times:
                self.stage_stats[stage] = {
                    'min': min(times),
                    'max': max(times),
                    'mean': statistics.mean(times),
                    'median': statistics.median(times),
                    'std_dev': statistics.stdev(times) if len(times) > 1 else 0
                }

    def plot_stage_distributions(self):
        """Создает графики распределения времени по этапам"""
        # Создаем DataFrame для удобства построения графиков
        data = []
        for stage, times in self.stage_times.items():
            for t in times:
                data.append({'Stage': stage, 'Time (s)': t})
        
        df = pd.DataFrame(data)
        
        # Создаем box plot
        plt.figure(figsize=(10, 6))
        sns.boxplot(x='Stage', y='Time (s)', data=df)
        plt.title('Distribution of Processing Times by Stage')
        plt.xticks(rotation=45)
        plt.tight_layout()
        plt.savefig('stage_times_distribution.png')
        plt.close()
        
        # Создаем violin plot
        plt.figure(figsize=(10, 6))
        sns.violinplot(x='Stage', y='Time (s)', data=df)
        plt.title('Violin Plot of Processing Times by Stage')
        plt.xticks(rotation=45)
        plt.tight_layout()
        plt.savefig('stage_times_violin.png')
        plt.close()

    def print_results(self):
        if not self.results:
            print("\nNo successful requests to analyze")
            if self.errors:
                print("\nAll requests failed with errors:")
                for error in self.errors:
                    print(error)
            return

        total_time = self.end_time - self.start_time
        successful_requests = len(self.results)
        failed_requests = len(self.errors)

        print("\n=== Load Test Results ===")
        print(f"Total time: {total_time:.2f} seconds")
        print(f"Total requests: {self.total_requests}")
        print(f"Successful requests: {successful_requests}")
        print(f"Failed requests: {failed_requests}")
        print(f"Requests per second: {successful_requests / total_time:.2f}")
        
        # Анализируем статистику по этапам
        self.analyze_stages()
        
        print("\n=== Processing Stage Statistics ===")
        for stage, stats in self.stage_stats.items():
            print(f"\n{stage.upper()} Times (seconds):")
            print(f"Min: {stats['min']:.3f}")
            print(f"Max: {stats['max']:.3f}")
            print(f"Mean: {stats['mean']:.3f}")
            print(f"Median: {stats['median']:.3f}")
            print(f"Std Dev: {stats['std_dev']:.3f}")
        
        if self.response_sizes:
            print(f"\nResponse size statistics (bytes):")
            print(f"Min: {min(self.response_sizes)}")
            print(f"Max: {max(self.response_sizes)}")
            print(f"Mean: {statistics.mean(self.response_sizes):.0f}")
            print(f"Median: {statistics.median(self.response_sizes):.0f}")
            print(f"Total data transferred: {sum(self.response_sizes) / (1024*1024):.2f} MB")

        if self.errors:
            print("\nErrors:")
            for error in self.errors[:10]:
                print(error)
            if len(self.errors) > 10:
                print(f"... and {len(self.errors) - 10} more errors")
        
        # Создаем визуализации
        self.plot_stage_distributions()
        print("\nGenerated visualization plots: stage_times_distribution.png and stage_times_violin.png")

def main():
    parser = argparse.ArgumentParser(description='Load testing tool for PDF Service')
    parser.add_argument('--url', default='http://172.27.239.31:31005/api/v1/docx',
                      help='Target URL (default: http://172.27.239.31:31005/api/v1/docx)')
    parser.add_argument('-c', '--concurrency', type=int, default=10,
                      help='Number of concurrent requests (default: 10)')
    parser.add_argument('-r', '--requests', type=int, default=100,
                      help='Total number of requests to make (default: 100)')
    parser.add_argument('--data', type=str, default='test-request.json',
                      help='JSON file with request data (default: test-request.json)')
    parser.add_argument('--output', type=str, default='load_test_results.json',
                      help='Output file for detailed results (default: load_test_results.json)')

    args = parser.parse_args()

    try:
        with open(args.data, 'r', encoding='utf-8') as f:
            request_data = json.load(f)
    except Exception as e:
        print(f"Error reading request data file: {e}")
        sys.exit(1)

    tester = LoadTester(args.url, args.concurrency, args.requests, request_data)
    
    print(f"\nStarting load test at {datetime.now()}")
    print(f"URL: {args.url}")
    print(f"Concurrency: {args.concurrency}")
    print(f"Total requests: {args.requests}")
    
    asyncio.run(tester.run_test())
    tester.print_results()
    
    # Сохраняем детальные результаты в JSON
    results = {
        'test_config': {
            'url': args.url,
            'concurrency': args.concurrency,
            'total_requests': args.requests,
            'timestamp': datetime.now().isoformat()
        },
        'stage_stats': tester.stage_stats,
        'errors': tester.errors,
        'response_sizes': {
            'min': min(tester.response_sizes) if tester.response_sizes else 0,
            'max': max(tester.response_sizes) if tester.response_sizes else 0,
            'mean': statistics.mean(tester.response_sizes) if tester.response_sizes else 0,
            'median': statistics.median(tester.response_sizes) if tester.response_sizes else 0
        }
    }
    
    with open(args.output, 'w', encoding='utf-8') as f:
        json.dump(results, f, indent=2)
    print(f"\nDetailed results saved to {args.output}")

if __name__ == '__main__':
    main() 