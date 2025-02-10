#!/usr/bin/env python3
import aiohttp
import asyncio
import json
import time
import argparse
from datetime import datetime
import statistics
import sys

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

    async def make_request(self, session, request_id):
        try:
            start_time = time.time()
            async with session.post(self.url, json=self.request_data) as response:
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
                    print(f"Request {request_id} completed in {response_time:.2f}s, response size: {len(content)} bytes")
                return status, response_time
        except Exception as e:
            error_msg = f"Request {request_id}: Exception: {str(e)}"
            print(f"Error on request {request_id}: {error_msg}")
            self.errors.append(error_msg)
            return None, None

    async def run_test(self):
        self.start_time = time.time()
        async with aiohttp.ClientSession() as session:
            tasks = []
            for i in range(self.total_requests):
                task = asyncio.create_task(self.make_request(session, i + 1))
                tasks.append(task)
                if len(tasks) >= self.concurrency:
                    await asyncio.gather(*tasks)
                    tasks = []
                    print(f"Completed batch of {self.concurrency} requests")
            if tasks:
                await asyncio.gather(*tasks)
        self.end_time = time.time()

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
        
        if self.results:
            print(f"\nResponse time statistics (seconds):")
            print(f"Min: {min(self.results):.3f}")
            print(f"Max: {max(self.results):.3f}")
            print(f"Mean: {statistics.mean(self.results):.3f}")
            print(f"Median: {statistics.median(self.results):.3f}")
            if len(self.results) > 1:
                print(f"Std Dev: {statistics.stdev(self.results):.3f}")

        if self.response_sizes:
            print(f"\nResponse size statistics (bytes):")
            print(f"Min: {min(self.response_sizes)}")
            print(f"Max: {max(self.response_sizes)}")
            print(f"Mean: {statistics.mean(self.response_sizes):.0f}")
            print(f"Median: {statistics.median(self.response_sizes):.0f}")
            print(f"Total data transferred: {sum(self.response_sizes) / (1024*1024):.2f} MB")

        if self.errors:
            print("\nErrors:")
            for error in self.errors[:10]:  # Show only first 10 errors
                print(error)
            if len(self.errors) > 10:
                print(f"... and {len(self.errors) - 10} more errors")

def main():
    parser = argparse.ArgumentParser(description='Load testing tool for PDF Service')
    parser.add_argument('--url', default='http://172.27.239.31:31005/generate-pdf',
                      help='Target URL (default: http://172.27.239.31:31005/generate-pdf)')
    parser.add_argument('--concurrency', type=int, default=10,
                      help='Number of concurrent requests (default: 10)')
    parser.add_argument('--requests', type=int, default=100,
                      help='Total number of requests to make (default: 100)')
    parser.add_argument('--data', type=str, default='test-request.json',
                      help='JSON file with request data (default: test-request.json)')

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

if __name__ == '__main__':
    main() 