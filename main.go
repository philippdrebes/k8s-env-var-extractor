package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	//v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type EnvVar struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	SlotSetting bool   `json:"slotSetting"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: ./main <directory>")
	}

	dir := os.Args[1]

	var deployments []appsv1.Deployment
	configs := make(map[string]corev1.ConfigMap)
	secrets := make(map[string]corev1.Secret)

	// Load all Kubernetes objects from all files in the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			docs := strings.Split(string(content), "---")
			for _, doc := range docs {
				// Try to unmarshal as Deployment
				var deploy appsv1.Deployment
				err := yaml.Unmarshal([]byte(doc), &deploy)
				if err == nil && deploy.TypeMeta.Kind == "Deployment" {
					deployments = append(deployments, deploy)
					continue
				}

				// Try to unmarshal as ConfigMap
				var config corev1.ConfigMap
				err = yaml.Unmarshal([]byte(doc), &config)
				if err == nil && config.TypeMeta.Kind == "ConfigMap" {
					configs[config.Name] = config
					continue
				}

				// Try to unmarshal as Secret
				var secret corev1.Secret
				err = yaml.Unmarshal([]byte(doc), &secret)
				if err == nil && secret.TypeMeta.Kind == "Secret" {
					secrets[secret.Name] = secret
					continue
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	for _, deploy := range deployments {
		envVars := make([]EnvVar, 0)

		for _, container := range deploy.Spec.Template.Spec.Containers {
			for _, env := range container.Env {
				var value string

				if env.Value != "" {
					value = env.Value
				} else if env.ValueFrom != nil {
					if env.ValueFrom.ConfigMapKeyRef != nil {
						config, ok := configs[env.ValueFrom.ConfigMapKeyRef.Name]
						if ok {
							value = config.Data[env.ValueFrom.ConfigMapKeyRef.Key]
						}
					} else if env.ValueFrom.SecretKeyRef != nil {
						secret, ok := secrets[env.ValueFrom.SecretKeyRef.Name]
						if ok {
							value = string(secret.Data[env.ValueFrom.SecretKeyRef.Key])
						}
					}
				}

				if value != "" {
					envVars = append(envVars, EnvVar{Name: env.Name, Value: value, SlotSetting: false})
				}
			}
		}

        if _, err := os.Stat("out"); os.IsNotExist(err) {
                os.MkdirAll("out", 0755)
        }

		if len(envVars) > 0 {
			envVarsJSON, err := json.MarshalIndent(envVars, "", "  ")
			if err != nil {
				log.Fatal("Error marshaling to JSON:", err)
			}

			fileName := fmt.Sprintf("out/%s.json", deploy.Name)
			err = ioutil.WriteFile(fileName, envVarsJSON, 0644)
			if err != nil {
				log.Fatal("Error writing JSON to file:", err)
			}
		}
	}
}

