package itests

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/logging"
)

const (
	dockerfilePath = "./../Dockerfile.gowaves-it"
	logFilePath    = "./../build/logs/go-node-container-build.log"
	errFilePath    = "./../build/logs/go-node-container-build.err"
)

const (
	keepDanglingEnvKey     = "ITESTS_KEEP_DANGLING"
	withRaceDetectorEnvKey = "ITESTS_WITH_RACE_DETECTOR"
)

const (
	withRaceDetectorSuffixArgumentName  = "WITH_RACE_SUFFIX"
	withRaceDetectorSuffixArgumentValue = "-with-race"
	versionArgumentName                 = "VERSION"
	itestsTmpImageLabel                 = "wavesplatform-gowaves-itests-tmp=true"
)

func TestMain(m *testing.M) {
	if err := testsSetup(); err != nil {
		slog.Error("Tests setup failed", logging.Error(err))
		os.Exit(1)
	}
	res := m.Run()
	//TODO: Add teardown if needed.
	os.Exit(res)
}

func testsSetup() error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	var (
		keepDangling     = mustBoolEnv(keepDanglingEnvKey)
		withRaceDetector = mustBoolEnv(withRaceDetectorEnvKey)
	)
	ctx := context.Background()
	// The API version is negotiated with Docker daemon, unless it's set with DOCKER_API_VERSION.
	cl, err := docker.NewClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if clErr := cl.Close(); clErr != nil {
			slog.Warn("Failed to close docker client", logging.Error(clErr))
		}
	}()

	platform := config.Platform()
	platformName := platform.OS + "/" + platform.Architecture
	slog.Info("Pulling scala-node image", "platform", platformName)
	scalaImage := config.ScalaImageRepository + ":" + config.ScalaImageTag
	if plErr := docker.PullImage(ctx, cl, scalaImage, platform); plErr != nil {
		return fmt.Errorf("failed to pull scala-node image: %w", plErr)
	}
	slog.Info("Building go-node image", "platform", platformName,
		"withRaceDetector", withRaceDetector)
	buildArgs := make(map[string]*string)
	if withRaceDetector {
		buildArgs[withRaceDetectorSuffixArgumentName] = new(withRaceDetectorSuffixArgumentValue)
	}
	if version := nodeVersion(); version != "" {
		buildArgs[versionArgumentName] = &version
	}
	dir, file := filepath.Split(filepath.Join(pwd, dockerfilePath))

	logFile, logCleanup, err := createLogFile(filepath.Join(pwd, logFilePath))
	if err != nil {
		return err
	}
	defer logCleanup()
	errFile, errCleanup, err := createLogFile(filepath.Join(pwd, errFilePath))
	if err != nil {
		return err
	}
	defer errCleanup()

	err = docker.BuildImage(ctx, cl, docker.BuildOptions{
		Name:       "go-node",
		ContextDir: dir,
		Dockerfile: file,
		BuildArgs:  buildArgs,
		Platform:   platform,
		Output:     logFile,
		ErrOutput:  errFile,
	})
	if err != nil {
		return fmt.Errorf("failed to build go-node image: %w", err)
	}

	if !keepDangling { // remove dangling images
		if rmErr := docker.RemoveImagesByLabel(ctx, cl, itestsTmpImageLabel); rmErr != nil {
			return rmErr
		}
	}
	return nil
}

// nodeVersion returns the version of the node built into the image. It is calculated on the host in the same way
// as in Makefile, because the build context passed to Docker contains no Git metadata. Empty result means that
// the version can't be detected, in that case the default value from Dockerfile is used.
func nodeVersion() string {
	out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		slog.Warn("Failed to detect node version", logging.Error(err))
		return ""
	}
	return strings.TrimSpace(string(out))
}

func createLogFile(path string) (*os.File, func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log file: %w", err)
	}
	cleanup := func() {
		if clErr := f.Close(); clErr != nil {
			slog.Warn("Failed to close file", slog.String("file", path), logging.Error(clErr))
		}
	}
	return f, cleanup, nil
}

func mustBoolEnv(key string) bool {
	val := os.Getenv(key)
	if val == "" {
		return false
	}
	r, err := strconv.ParseBool(val)
	if err != nil {
		slog.Error("Environment variable has not a boolean value", slog.String("variable", key),
			slog.Any("value", val), logging.Error(err))
	}
	return r
}
