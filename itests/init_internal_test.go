package itests

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ory/dockertest/v3"
	dc "github.com/ory/dockertest/v3/docker"

	"github.com/wavesplatform/gowaves/itests/config"
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
	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("failed to connect to docker: %w", err)
	}

	platform := config.Platform()
	slog.Info("Pulling scala-node image", "platform", platform)
	if plErr := pool.Client.PullImage(
		dc.PullImageOptions{
			Repository: "wavesplatform/wavesnode",
			Tag:        "latest",
			Platform:   platform,
		},
		dc.AuthConfiguration{}); plErr != nil {
		return fmt.Errorf("failed to pull scala-node image: %w", plErr)
	}
	slog.Info("Building go-node image", "platform", platform, "withRaceDetector", withRaceDetector)
	var buildArgs []dc.BuildArg
	if withRaceDetector {
		buildArgs = append(buildArgs, dc.BuildArg{
			Name: withRaceDetectorSuffixArgumentName, Value: withRaceDetectorSuffixArgumentValue,
		})
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

	err = pool.Client.BuildImage(dc.BuildImageOptions{
		Name:           "go-node",
		Dockerfile:     file,
		ContextDir:     dir,
		OutputStream:   logFile,
		ErrorStream:    errFile,
		BuildArgs:      buildArgs,
		Platform:       platform,
		RmTmpContainer: true,
	})
	if err != nil {
		return fmt.Errorf("failed to build go-node image: %w", err)
	}

	if !keepDangling { // remove dangling images
		images, lsErr := pool.Client.ListImages(dc.ListImagesOptions{
			Filters: map[string][]string{
				"label": {"wavesplatform-gowaves-itests-tmp=true"},
			},
		})
		if lsErr != nil {
			return fmt.Errorf("failed to list images: %w", lsErr)
		}
		for _, i := range images {
			rmErr := pool.Client.RemoveImageExtended(i.ID, dc.RemoveImageOptions{
				Force:   true,
				NoPrune: false,
				Context: nil,
			})
			if rmErr != nil {
				return fmt.Errorf("failed to remove image: %w", rmErr)
			}
		}
	}
	return nil
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
