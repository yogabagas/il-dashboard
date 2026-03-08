#!/usr/bin/env python3
"""
Simple JSON log viewer - No jq needed!
Usage: python3 view_logs.py [options]
"""

import json
import sys
import glob
from datetime import datetime

def print_log_entry(entry, pretty=True):
    """Print a single log entry"""
    if pretty:
        timestamp = entry.get('timestamp', 'N/A')
        level = entry.get('level', 'INFO').upper()
        message = entry.get('message', '')
        
        # Color codes
        colors = {
            'ERROR': '\033[91m',
            'WARN': '\033[93m',
            'INFO': '\033[92m',
            'DEBUG': '\033[94m',
            'RESET': '\033[0m'
        }
        
        color = colors.get(level.upper(), colors['INFO'])
        reset = colors['RESET']
        
        print(f"{color}[{timestamp}] {level:5s}{reset} {message}")
        
        # Print data fields if present
        if 'data' in entry and entry['data']:
            for key, value in entry['data'].items():
                print(f"  {key}: {value}")
        
        # Print caller info if present
        if 'caller' in entry and entry['caller']:
            caller = entry['caller']
            file = caller.get('file', 'unknown')
            line = caller.get('line', '?')
            print(f"  @ {file}:{line}")
        
        print()  # Empty line
    else:
        # Print raw JSON
        print(json.dumps(entry, indent=2))

def view_logs(filter_level=None, filter_component=None, limit=None):
    """View JSON logs with optional filtering"""
    # Find all JSON log files
    log_files = sorted(glob.glob('logs/il-dashboard/app-*.json'), reverse=True)
    
    if not log_files:
        print("No log files found in logs/il-dashboard/")
        return
    
    print(f"Found {len(log_files)} log file(s)")
    print(f"Reading: {log_files[0]}")
    print("-" * 80)
    print()
    
    count = 0
    with open(log_files[0], 'r') as f:
        for line in f:
            if not line.strip():
                continue
            
            try:
                entry = json.loads(line)
                
                # Apply filters
                if filter_level and entry.get('level') != filter_level.lower():
                    continue
                
                if filter_component and entry.get('data', {}).get('component') != filter_component:
                    continue
                
                print_log_entry(entry)
                count += 1
                
                if limit and count >= limit:
                    break
                    
            except json.JSONDecodeError:
                print(f"Warning: Skipped invalid JSON line")
    
    print("-" * 80)
    print(f"Displayed {count} log entries")

def tail_logs(lines=20):
    """Show last N lines from log file"""
    log_files = sorted(glob.glob('logs/il-dashboard/app-*.json'), reverse=True)
    
    if not log_files:
        print("No log files found in logs/il-dashboard/")
        return
    
    print(f"Last {lines} entries from: {log_files[0]}")
    print("-" * 80)
    print()
    
    with open(log_files[0], 'r') as f:
        all_lines = f.readlines()
        for line in all_lines[-lines:]:
            if line.strip():
                try:
                    entry = json.loads(line)
                    print_log_entry(entry)
                except json.JSONDecodeError:
                    print(f"Warning: Skipped invalid JSON line")

def search_logs(query):
    """Search for text in logs"""
    log_files = sorted(glob.glob('logs/il-dashboard/app-*.json'), reverse=True)
    
    if not log_files:
        print("No log files found in logs/il-dashboard/")
        return
    
    print(f"Searching for: '{query}'")
    print("-" * 80)
    print()
    
    count = 0
    with open(log_files[0], 'r') as f:
        for line in f:
            if query.lower() in line.lower():
                try:
                    entry = json.loads(line)
                    print_log_entry(entry)
                    count += 1
                except json.JSONDecodeError:
                    print(f"Warning: Skipped invalid JSON line")
    
    print("-" * 80)
    print(f"Found {count} matching entries")

if __name__ == '__main__':
    import argparse
    
    parser = argparse.ArgumentParser(description='View JSON logs')
    parser.add_argument('--level', choices=['debug', 'info', 'warn', 'error'], help='Filter by log level')
    parser.add_argument('--component', help='Filter by component name')
    parser.add_argument('--limit', type=int, help='Limit number of entries to show')
    parser.add_argument('--tail', type=int, default=0, help='Show last N entries')
    parser.add_argument('--search', help='Search for text in logs')
    
    args = parser.parse_args()
    
    if args.search:
        search_logs(args.search)
    elif args.tail > 0:
        tail_logs(args.tail)
    else:
        view_logs(args.level, args.component, args.limit)
