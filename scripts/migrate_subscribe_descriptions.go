package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxysubscribe"
	"github.com/npanel-dev/NPanel-backend/internal/database"
)

type feature struct {
	Icon  string `json:"icon"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type legacyObject struct {
	Description   string          `json:"description"`
	Features      json.RawMessage `json:"features"`
	DetailFormat  string          `json:"detail_format"`
	DetailContent string          `json:"detail_content"`
	Content       string          `json:"content"`
}

func main() {
	var dsn string
	var driverName string
	var apply bool
	flag.StringVar(&driverName, "driver", envOrDefault("NPANEL_DATABASE_DRIVER", "mysql"), "Database driver: mysql or postgres. Defaults to NPANEL_DATABASE_DRIVER.")
	flag.StringVar(&dsn, "dsn", os.Getenv("NPANEL_DATABASE_DSN"), "Database DSN. Defaults to NPANEL_DATABASE_DSN.")
	flag.BoolVar(&apply, "apply", false, "Write changes. Without this flag the script only prints a dry-run summary.")
	flag.Parse()

	if strings.TrimSpace(dsn) == "" {
		log.Fatal("missing DSN: pass --dsn or set NPANEL_DATABASE_DSN")
	}

	driver, err := database.NormalizeDriver(driverName)
	if err != nil {
		log.Fatal(err)
	}
	client, err := database.OpenEnt(driver, dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	items, err := client.ProxySubscribe.Query().Order(ent.Asc(proxysubscribe.FieldID)).All(ctx)
	if err != nil {
		log.Fatalf("query subscribes: %v", err)
	}

	var changed int
	for _, item := range items {
		if item.Description == nil || strings.TrimSpace(*item.Description) == "" {
			continue
		}
		if hasNewDescriptionFields(item) {
			continue
		}

		shortDescription, featuresJSON, detailFormat, detailContent := migrateDescription(*item.Description)
		if shortDescription == "" && featuresJSON == "" && detailContent == "" {
			continue
		}

		changed++
		fmt.Printf("subscribe #%d %q -> short=%q features=%dB format=%s detail=%dB\n",
			item.ID,
			item.Name,
			shortDescription,
			len(featuresJSON),
			detailFormat,
			len(detailContent),
		)

		if !apply {
			continue
		}

		if err := client.ProxySubscribe.UpdateOneID(item.ID).
			SetShortDescription(shortDescription).
			SetFeatures(featuresJSON).
			SetDetailFormat(detailFormat).
			SetDetailContent(detailContent).
			Exec(ctx); err != nil {
			log.Fatalf("update subscribe #%d: %v", item.ID, err)
		}
	}

	if apply {
		fmt.Printf("updated %d subscribe descriptions\n", changed)
	} else {
		fmt.Printf("dry-run found %d subscribe descriptions to migrate; rerun with --apply to write changes\n", changed)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func hasNewDescriptionFields(item *ent.ProxySubscribe) bool {
	return stringPointerValue(item.ShortDescription) != "" ||
		stringPointerValue(item.Features) != "" ||
		stringPointerValue(item.DetailContent) != ""
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func migrateDescription(raw string) (string, string, string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", "markdown", ""
	}

	var object legacyObject
	if err := json.Unmarshal([]byte(trimmed), &object); err == nil && (object.Description != "" || len(object.Features) > 0 || object.Content != "" || object.DetailContent != "") {
		features := normalizeFeatures(object.Features)
		return strings.TrimSpace(object.Description), marshalFeatures(features), normalizeDetailFormat(object.DetailFormat), strings.TrimSpace(firstNonEmpty(object.DetailContent, object.Content))
	}

	var array []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &array); err == nil {
		return "", marshalFeatures(normalizeFeatureArray(array)), "markdown", ""
	}

	return trimmed, "", "text", trimmed
}

func normalizeFeatures(raw json.RawMessage) []feature {
	if len(raw) == 0 {
		return nil
	}
	var array []map[string]any
	if err := json.Unmarshal(raw, &array); err != nil {
		return nil
	}
	return normalizeFeatureArray(array)
}

func normalizeFeatureArray(array []map[string]any) []feature {
	features := make([]feature, 0, len(array))
	for _, item := range array {
		label := firstNonEmpty(fmt.Sprint(item["label"]), fmt.Sprint(item["feature"]), fmt.Sprint(item["text"]))
		if label == "" || label == "<nil>" {
			continue
		}
		featureType := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["type"])))
		if featureType != "success" && featureType != "destructive" && featureType != "default" {
			if support, ok := item["support"].(bool); ok && support {
				featureType = "success"
			} else if ok && !support {
				featureType = "destructive"
			} else {
				featureType = "default"
			}
		}
		icon := ""
		if value, ok := item["icon"].(string); ok {
			icon = value
		}
		features = append(features, feature{
			Icon:  icon,
			Label: label,
			Type:  featureType,
		})
	}
	return features
}

func marshalFeatures(features []feature) string {
	if len(features) == 0 {
		return ""
	}
	data, err := json.Marshal(features)
	if err != nil {
		return ""
	}
	return string(data)
}

func normalizeDetailFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "html":
		return "html"
	case "text", "plain":
		return "text"
	default:
		return "markdown"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && trimmed != "<nil>" {
			return trimmed
		}
	}
	return ""
}
