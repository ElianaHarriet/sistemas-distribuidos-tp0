#!/usr/bin/env python3

import yaml
import argparse

def server_service():
    return {
        'container_name': 'server',
        'image': 'server:latest',
        'entrypoint': 'python3 /main.py',
        'environment': [
            'PYTHONUNBUFFERED=1',
            'LOGGING_LEVEL=DEBUG'
        ],
        'networks': ['testing_net']
    }

def client_service(id):
    return {
        'container_name': f'client{id}',
        'image': 'client:latest',
        'entrypoint': '/client',
        'environment': [
            f'CLI_ID={id}',
            'CLI_LOG_LEVEL=DEBUG'
        ],
        'networks': ['testing_net'],
        'depends_on': ['server']
    }

def generate_services(num_clients):
    services = {'server': server_service()}
    for i in range(1, num_clients + 1):
        services[f'client{i}'] = client_service(i)
    return services

def generate_docker_compose(num_clients):
    docker_compose = {
        'version': '3.9',
        'name': 'tp0',
        'services': generate_services(num_clients),
        'networks': {
            'testing_net': {
                'ipam': {
                    'driver': 'default',
                    'config': [{'subnet': '172.25.125.0/24'}]
                }
            }
        }
    }

    with open('docker-compose-dev1.1.yaml', 'w') as file:
        yaml.dump(docker_compose, file, default_flow_style=False, sort_keys=False)

def main():
    parser = argparse.ArgumentParser(description='Generates a docker-compose.yaml file with a configurable number of clients.')
    parser.add_argument('num_clients', type=int, help='The number of clients to generate.')

    args = parser.parse_args()

    generate_docker_compose(args.num_clients)

if __name__ == '__main__':
    main()