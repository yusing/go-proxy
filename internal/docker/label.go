package docker

import (
	"strings"

	E "github.com/yusing/go-proxy/internal/error"
)

type LabelMap = map[string]any

func ParseLabels(labels map[string]string) (LabelMap, E.Error) {
	nestedMap := make(LabelMap)
	errs := E.NewBuilder("labels error")

	for lbl, value := range labels {
		parts := strings.Split(lbl, ".")
		if parts[0] != NSProxy {
			continue
		}
		if len(parts) == 1 {
			errs.Add(E.Errorf("invalid label %s", lbl).Subject(lbl))
			continue
		}
		parts = parts[1:]
		currentMap := nestedMap

		for i, k := range parts {
			if i == len(parts)-1 {
				// Last element, set the value
				currentMap[k] = value
			} else {
				// If the key doesn't exist, create a new map
				if _, exists := currentMap[k]; !exists {
					currentMap[k] = make(LabelMap)
				}
				// Move deeper into the nested map
				m, ok := currentMap[k].(LabelMap)
				if !ok && currentMap[k] != "" {
					errs.Add(E.Errorf("expect mapping, got %T", currentMap[k]).Subject(lbl))
					continue
				} else if !ok {
					m = make(LabelMap)
					currentMap[k] = m
				}
				currentMap = m
			}
		}
	}

	return nestedMap, errs.Error()
}
