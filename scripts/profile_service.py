#!/usr/bin/env python3
import argparse
import subprocess
import time
import os
import signal
import sys
from datetime import datetime

def run_command(command):
    try:
        return subprocess.run(command, shell=True, check=True, capture_output=True, text=True)
    except subprocess.CalledProcessError as e:
        print(f"Error executing command: {command}")
        print(f"Error output: {e.stderr}")
        return None

def setup_port_forward(pod_name, pprof_port):
    print(f"Setting up port-forward for pprof on port {pprof_port}...")
    cmd = f"kubectl port-forward -n print-serv {pod_name} {pprof_port}:6060"
    process = subprocess.Popen(cmd, shell=True)
    time.sleep(5)  # Wait for port-forward to establish
    return process

def collect_profile(profile_type, duration, output_dir, pprof_port):
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    profile_file = f"{output_dir}/{profile_type}_{timestamp}.prof"
    
    if profile_type == "cpu":
        url = f"http://localhost:{pprof_port}/debug/pprof/profile?seconds={duration}"
    elif profile_type == "heap":
        url = f"http://localhost:{pprof_port}/debug/pprof/heap"
    elif profile_type == "goroutine":
        url = f"http://localhost:{pprof_port}/debug/pprof/goroutine"
    else:
        print(f"Unknown profile type: {profile_type}")
        return None

    print(f"Collecting {profile_type} profile...")
    cmd = f"curl -o {profile_file} {url}"
    result = run_command(cmd)
    
    if result and result.returncode == 0:
        print(f"Profile saved to {profile_file}")
        return profile_file
    return None

def analyze_profile(profile_file, profile_type, output_dir):
    if not profile_file or not os.path.exists(profile_file):
        return

    base_name = os.path.splitext(os.path.basename(profile_file))[0]
    
    # Generate text report
    text_file = f"{output_dir}/{base_name}.txt"
    run_command(f"go tool pprof -text {profile_file} > {text_file}")
    
    # Generate graph
    graph_file = f"{output_dir}/{base_name}.png"
    run_command(f"go tool pprof -png {profile_file} > {graph_file}")
    
    # For heap profiles, generate both alloc and inuse space analysis
    if profile_type == "heap":
        run_command(f"go tool pprof -alloc_space -text {profile_file} > {output_dir}/{base_name}_alloc.txt")
        run_command(f"go tool pprof -inuse_space -text {profile_file} > {output_dir}/{base_name}_inuse.txt")

def main():
    parser = argparse.ArgumentParser(description='Profile PDF service in Kubernetes')
    parser.add_argument('--pod-name', required=True, help='Name of the pod to profile')
    parser.add_argument('--duration', type=int, default=60, help='Duration of profiling in seconds')
    parser.add_argument('--pprof-port', type=int, default=6060, help='Local port for pprof')
    parser.add_argument('--output-dir', default='profiles', help='Output directory for profiles')
    parser.add_argument('--load-test', action='store_true', help='Run load test during profiling')
    parser.add_argument('--load-test-concurrency', type=int, default=10, help='Load test concurrency')
    parser.add_argument('--load-test-requests', type=int, default=100, help='Number of load test requests')
    
    args = parser.parse_args()

    # Create output directory
    os.makedirs(args.output_dir, exist_ok=True)

    # Setup port forwarding
    port_forward_process = setup_port_forward(args.pod_name, args.pprof_port)

    try:
        # Start load test if requested
        load_test_process = None
        if args.load_test:
            print("Starting load test...")
            load_test_cmd = f"python scripts/load_test.py -c {args.load_test_concurrency} -r {args.load_test_requests}"
            load_test_process = subprocess.Popen(load_test_cmd, shell=True)

        # Collect profiles
        profiles = []
        for profile_type in ["cpu", "heap", "goroutine"]:
            profile_file = collect_profile(profile_type, args.duration, args.output_dir, args.pprof_port)
            if profile_file:
                profiles.append((profile_file, profile_type))

        # Wait for load test to complete if it was started
        if load_test_process:
            load_test_process.wait()

        # Analyze collected profiles
        print("\nAnalyzing profiles...")
        for profile_file, profile_type in profiles:
            analyze_profile(profile_file, profile_type, args.output_dir)

    finally:
        # Cleanup
        if port_forward_process:
            port_forward_process.terminate()
            port_forward_process.wait()

    print(f"\nProfiling completed. Results are in {args.output_dir}/")

if __name__ == '__main__':
    main() 