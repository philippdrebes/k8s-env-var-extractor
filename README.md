# Kubernetes Environment Variable Extractor
This is a simple Go project that extracts environment variables from Kubernetes Deployments, ConfigMaps and Secrets YAML files. It generates a JSON file for each deployment, which can be imported into Azure app configuraations. 

## Prerequisites

- Go (v1.20 or later)
- Kubernetes API libraries (`k8s.io/api v0.27.3`)
- YAML library for Go (`sigs.k8s.io/yaml v1.3.0`)

## Installation

The project uses Go Modules for dependency management.

```bash
git clone https://github.com/philippdrebes/k8s-env-var-extractor.git
cd k8s-env-var-extractor
go build
```

## Usage
After building the project, you can run the program by specifying the directory path that contains your Kubernetes YAML files:

```bash
./k8s-env-var-extractor <directory>
```

The program will recursively scan the specified directory and its subdirectories for YAML files.

For each Deployment, it will create a JSON file named after the Deployment in the out directory. 
The JSON file contains the environment variables for the corresponding Deployment. 
If an environment variable references a ConfigMap or Secret, the program resolves the reference and includes the actual value in the JSON output.

## Built With
- [Go](https://go.dev/)
- [Kubernetes API libraries](https://github.com/kubernetes/api)
- [YAML library for Go](https://github.com/kubernetes-sigs/yaml)

## Author
- Philipp Drebes
