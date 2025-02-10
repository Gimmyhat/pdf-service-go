#!/usr/bin/env python3
import aiohttp
import asyncio
import json
import time
import argparse
from datetime import datetime
import statistics
import sys
import random
from aiohttp import ClientTimeout

class RetryTester:
    def __init__(self, url, request_data):
        self.url = url
        self.request_data = request_data
        self.results = {
            'timeout': [],
            'connection': [],
            'validation_error': [],
            'server_error': [],
            'success': []
        }
        self.test_scenarios = [
            ('timeout', self.test_timeout),
            ('connection', self.test_connection_error),
            ('validation_error', self.test_validation_error),
            ('server_error', self.test_server_error),
            ('success', self.test_success)
        ]

    async def test_timeout(self, session):
        """Тест таймаутов"""
        try:
            timeout = ClientTimeout(total=0.1)  # Очень маленький таймаут
            async with session.post(self.url, json=self.request_data, timeout=timeout) as response:
                return 'timeout', response.status
        except asyncio.TimeoutError:
            return 'timeout', 'timeout'
        except Exception as e:
            return 'timeout', str(e)

    async def test_connection_error(self, session):
        """Тест ошибок соединения"""
        bad_url = self.url.replace('http://', 'http://invalid.')
        try:
            async with session.post(bad_url, json=self.request_data) as response:
                return 'connection', response.status
        except Exception as e:
            return 'connection', str(e)

    async def test_validation_error(self, session):
        """Тест ошибок валидации"""
        error_cases = [
            # Невалидный JSON
            ('Invalid JSON', b'{"this is not valid json'),
            # Отсутствуют обязательные поля
            ('Missing required fields', {'operation': 'CREATE'}),
            # Неверный формат данных
            ('Invalid data format', {**self.request_data, 'creationDate': 'not-a-date'})
        ]
        
        for error_name, error_data in error_cases:
            try:
                headers = {'Content-Type': 'application/json'}
                if isinstance(error_data, bytes):
                    async with session.post(self.url, data=error_data, headers=headers) as response:
                        return 'validation_error', f"{error_name}: {response.status}"
                else:
                    async with session.post(self.url, json=error_data, headers=headers) as response:
                        return 'validation_error', f"{error_name}: {response.status}"
            except Exception as e:
                return 'validation_error', f"{error_name}: {str(e)}"

    async def test_server_error(self, session):
        """Тест серверных ошибок"""
        error_cases = [
            # Некорректная операция
            {**self.request_data, 'operation': 'INVALID_OPERATION'},
            # Некорректный template_id
            {**self.request_data, 'registryItems': [{'id': 999999999, 'name': 'Non-existent template'}]},
            # Слишком большой запрос
            {**self.request_data, 'purposeOfGeoInfoAccess': 'x' * 1000000}
        ]
        
        for error_data in error_cases:
            try:
                async with session.post(self.url, json=error_data) as response:
                    content = await response.text()
                    return 'server_error', f"{response.status}: {content[:100]}"  # Берем только первые 100 символов ответа
            except Exception as e:
                return 'server_error', str(e)

    async def test_success(self, session):
        """Тест успешного запроса"""
        try:
            async with session.post(self.url, json=self.request_data) as response:
                return 'success', response.status
        except Exception as e:
            return 'success', str(e)

    async def run_test(self):
        print(f"\nStarting retry mechanism test at {datetime.now()}")
        
        async with aiohttp.ClientSession() as session:
            for scenario_name, test_func in self.test_scenarios:
                print(f"\nTesting {scenario_name} scenario:")
                for i in range(3):  # 3 попытки для каждого сценария
                    start_time = time.time()
                    result_type, status = await test_func(session)
                    duration = time.time() - start_time
                    
                    self.results[result_type].append({
                        'attempt': i + 1,
                        'status': status,
                        'duration': duration
                    })
                    
                    print(f"Attempt {i + 1}: Status: {status}, Duration: {duration:.2f}s")
                    await asyncio.sleep(2)  # Увеличенная пауза между запросами

    def print_results(self):
        print("\n=== Retry Mechanism Test Results ===")
        
        for scenario_type, results in self.results.items():
            if results:
                print(f"\n{scenario_type.upper()} Scenario Results:")
                print(f"Total attempts: {len(results)}")
                
                durations = [r['duration'] for r in results]
                statuses = [str(r['status']) for r in results]
                
                print(f"Response time statistics (seconds):")
                print(f"Min: {min(durations):.3f}")
                print(f"Max: {max(durations):.3f}")
                print(f"Mean: {statistics.mean(durations):.3f}")
                print(f"Statuses: {', '.join(statuses)}")

def main():
    parser = argparse.ArgumentParser(description='Retry mechanism testing tool for PDF Service')
    parser.add_argument('--url', default='http://172.27.239.31:31005/api/v1/docx',
                      help='Target URL (default: http://172.27.239.31:31005/api/v1/docx)')
    parser.add_argument('--data', type=str, default='test-request.json',
                      help='JSON file with request data (default: test-request.json)')

    args = parser.parse_args()

    try:
        with open(args.data, 'r', encoding='utf-8') as f:
            request_data = json.load(f)
    except Exception as e:
        print(f"Error reading request data file: {e}")
        sys.exit(1)

    tester = RetryTester(args.url, request_data)
    asyncio.run(tester.run_test())
    tester.print_results()

if __name__ == '__main__':
    main() 