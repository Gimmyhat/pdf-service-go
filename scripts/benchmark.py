#!/usr/bin/env python3
import subprocess
import time
import statistics

def run_command(cmd):
    start_time = time.time()
    process = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    end_time = time.time()
    return end_time - start_time, process.returncode == 0, process.stdout

def benchmark_go(iterations=10):  # Увеличим количество итераций для более точных результатов
    times = []
    successes = 0
    
    print(f"\nRunning benchmark for Go implementation")
    print("-" * 50)
    
    cmd = "go run cmd/test_docx/main.go"
    
    # Прогрев (первый запуск может быть медленнее)
    print("\nWarm-up run...")
    run_command(cmd)
    
    print("\nBenchmark runs:")
    for i in range(iterations):
        duration, success, output = run_command(cmd)
        if success:
            times.append(duration)
            successes += 1
        print(f"Iteration {i+1}: {duration:.3f} seconds {'✓' if success else '✗'}")
    
    if times:
        avg_time = statistics.mean(times)
        min_time = min(times)
        max_time = max(times)
        std_dev = statistics.stdev(times) if len(times) > 1 else 0
        
        print(f"\nResults:")
        print(f"Average time:   {avg_time:.3f} seconds")
        print(f"Minimum time:   {min_time:.3f} seconds")
        print(f"Maximum time:   {max_time:.3f} seconds")
        print(f"Std deviation:  {std_dev:.3f} seconds")
        print(f"Success rate:   {successes}/{iterations}")
    else:
        print("\nNo successful runs to analyze")
    
    return times

def main():
    print("Starting Go implementation performance test...")
    benchmark_go()

if __name__ == "__main__":
    main() 