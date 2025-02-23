package xconfig

import (
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/raphoester/x/basicutil"
)

func LoadDefaultEnvFile() {
	LoadEnvFile(".env")
}

func LoadEnvFile(name string) {
	projectRoot, _ := basicutil.FindProjectRoot()
	name = filepath.Join(projectRoot, name)
	_ = godotenv.Load(name)
}
