package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"github.com/alexflint/go-arg"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type EnvVar struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	SlotSetting bool   `json:"slotSetting"`
}

func main() {
	var args struct {
		Input  string `arg:"positional,required" placeholder:"SRC" help:"input dir containing YAML files"`
		Output string `arg:"positional" placeholder:"OUT" help:"output dir for JSON files" default:"./out"`
	}

	arg.MustParse(&args)

	var deployments []appsv1.Deployment
	configs := make(map[string]corev1.ConfigMap)
	secrets := make(map[string]corev1.Secret)

	err := filepath.Walk(args.Input, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			content, err := os.ReadFile(path)
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

	// Check if the "out" directory exists, if not create one
	if _, err := os.Stat(args.Output); os.IsNotExist(err) {
		err = os.MkdirAll(args.Output, 0755)
		if err != nil {
			log.Fatal(err)
		}
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
							value = string(secret.StringData[env.ValueFrom.SecretKeyRef.Key])
						}
					}
				}

				if value != "" {
					envVars = append(envVars, EnvVar{Name: env.Name, Value: value, SlotSetting: false})
				}
			}

			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					config, ok := configs[envFrom.ConfigMapRef.Name]
					if ok {
						for k, v := range config.Data {
							envVars = append(envVars, EnvVar{Name: k, Value: v, SlotSetting: false})
						}
					}
				} else if envFrom.SecretRef != nil {
					secret, ok := secrets[envFrom.SecretRef.Name]
					if ok {
						for k, v := range secret.StringData {
							envVars = append(envVars, EnvVar{Name: k, Value: string(v), SlotSetting: false})
						}
					}
				}
			}
		}

		slices.SortFunc(envVars, func(a, b EnvVar) int {
			return cmp.Compare(a.Name, b.Name)
		})

		if len(envVars) > 0 {
			envVarsJSON, err := json.MarshalIndent(envVars, "", "  ")
			if err != nil {
				log.Fatal("Error marshaling to JSON:", err)
			}

			fileName := fmt.Sprintf("out/%s.json", deploy.Name)
			err = os.WriteFile(fileName, envVarsJSON, 0644)
			if err != nil {
				log.Fatal("Error writing JSON to file:", err)
			}
		}
	}
}
