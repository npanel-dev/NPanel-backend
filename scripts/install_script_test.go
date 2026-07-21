package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runInstallBash(t *testing.T, script, tempDir string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", "-c", script, "bash", tempDir)
	cmd.Env = append(os.Environ(), "NPANEL_INSTALL_SOURCE_ONLY=1")
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestInstallScriptGeneratesBothDatabaseStacks(t *testing.T) {
	tempDir := t.TempDir()
	output, err := runInstallBash(t, `
source ./install.sh
root="$1"
VERSION=v1.0.9
INSTALL_DIR=/opt/npanel-backend
FORCE_CONFIG=1
for driver in mysql postgres; do
  DATABASE_DRIVER="$driver"
  DATABASE_DRIVER_SET=1
  CONFIG_DIR="$root/$driver"
  CONFIG_FILE="$CONFIG_DIR/config.yaml"
  ENV_FILE="$CONFIG_DIR/.env"
  COMPOSE_FILE="$CONFIG_DIR/docker-compose.yml"
  mkdir -p "$CONFIG_DIR"
  write_runtime_env >/dev/null
  write_docker_config >/dev/null
  write_compose_file >/dev/null
done
`, tempDir)
	if err != nil {
		t.Fatalf("generate installer configs: %v\n%s", err, output)
	}

	tests := []struct {
		driver       string
		serviceImage string
		dsnFragment  string
	}{
		{"mysql", "image: mysql:8.4", "@tcp(mysql:3306)/npanel"},
		{"postgres", "image: postgres:17", "@postgres:5432/npanel?sslmode=disable"},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			config := mustReadTestFile(t, filepath.Join(tempDir, tt.driver, "config.yaml"))
			env := mustReadTestFile(t, filepath.Join(tempDir, tt.driver, ".env"))
			compose := mustReadTestFile(t, filepath.Join(tempDir, tt.driver, "docker-compose.yml"))
			for name, check := range map[string]struct {
				content  string
				expected string
			}{
				"config driver": {config, "driver: " + tt.driver},
				"config DSN":    {config, tt.dsnFragment},
				"environment":   {env, "NPANEL_DATABASE_DRIVER=" + tt.driver},
				"compose image": {compose, tt.serviceImage},
				"dependency":    {compose, "      " + tt.driver + ":"},
			} {
				if !strings.Contains(check.content, check.expected) {
					t.Errorf("%s does not contain %q", name, check.expected)
				}
			}
		})
	}
}

func TestInstallScriptProtectsExistingDatabaseSelection(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "legacy")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte("MYSQL_DATABASE=npanel\nMYSQL_USER=npanel\nMYSQL_PASSWORD=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("data:\n  database:\n    driver: mysql\n    source: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := runInstallBash(t, `
source ./install.sh
CONFIG_DIR="$1/legacy"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
ENV_FILE="$CONFIG_DIR/.env"
DATABASE_DRIVER=postgres
DATABASE_DRIVER_SET=1
FORCE_CONFIG=0
write_runtime_env
`, tempDir)
	if err == nil {
		t.Fatalf("database switch without --force-config unexpectedly succeeded:\n%s", output)
	}
	if !strings.Contains(output, "已有环境变量使用 mysql") {
		t.Fatalf("unexpected mismatch error:\n%s", output)
	}

	output, err = runInstallBash(t, `
source ./install.sh
CONFIG_DIR="$1/legacy"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
ENV_FILE="$CONFIG_DIR/.env"
DATABASE_DRIVER=postgres
DATABASE_DRIVER_SET=""
FORCE_CONFIG=0
write_runtime_env >/dev/null
printf '%s' "$DATABASE_DRIVER"
`, tempDir)
	if err != nil {
		t.Fatalf("infer legacy database driver: %v\n%s", err, output)
	}
	if output != "mysql" {
		t.Fatalf("inferred driver = %q, want mysql", output)
	}
}

func TestInstallScriptForceConfigPreservesExistingCredentials(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "existing")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := strings.Join([]string{
		"NPANEL_DATABASE_DRIVER=mysql",
		"MYSQL_ROOT_PASSWORD=existing-root",
		"MYSQL_DATABASE=npanel",
		"MYSQL_USER=npanel",
		"MYSQL_PASSWORD=existing-database",
		"REDIS_PASSWORD=existing-redis",
		"JWT_SECRET=existing-jwt",
		"ADMIN_PASSWORD=existing-admin",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	output, err := runInstallBash(t, `
source ./install.sh
CONFIG_DIR="$1/existing"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
ENV_FILE="$CONFIG_DIR/.env"
DATABASE_DRIVER=mysql
DATABASE_DRIVER_SET=1
FORCE_CONFIG=1
VERSION=v1.0.9
write_runtime_env >/dev/null
`, tempDir)
	if err != nil {
		t.Fatalf("force config: %v\n%s", err, output)
	}
	env := mustReadTestFile(t, filepath.Join(configDir, ".env"))
	for _, expected := range []string{
		"MYSQL_ROOT_PASSWORD=existing-root",
		"MYSQL_PASSWORD=existing-database",
		"REDIS_PASSWORD=existing-redis",
		"JWT_SECRET=existing-jwt",
		"ADMIN_PASSWORD=existing-admin",
	} {
		if !strings.Contains(env, expected) {
			t.Errorf("rewritten environment does not contain %q", expected)
		}
	}
}

func TestDockerUpgradeUsesSelectedComposeFile(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "compose.log")
	fakeCompose := filepath.Join(tempDir, "fake-compose")
	fakeScript := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"$FAKE_COMPOSE_LOG\"\n"
	if err := os.WriteFile(fakeCompose, []byte(fakeScript), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", "./docker-upgrade.sh")
	cmd.Env = append(os.Environ(),
		"SKIP_GIT_PULL=1",
		"NPANEL_COMPOSE_FILE=docker-compose.postgres.yml",
		"DOCKER_COMPOSE="+fakeCompose,
		"FAKE_COMPOSE_LOG="+logPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker upgrade: %v\n%s", err, output)
	}
	lines := strings.FieldsFunc(mustReadTestFile(t, logPath), func(r rune) bool { return r == '\n' })
	if len(lines) != 3 {
		t.Fatalf("compose invocation count = %d, want 3: %q", len(lines), lines)
	}
	for _, line := range lines {
		if !strings.Contains(line, "-f docker-compose.postgres.yml") {
			t.Errorf("compose invocation does not use PostgreSQL file: %q", line)
		}
	}
}

func mustReadTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
