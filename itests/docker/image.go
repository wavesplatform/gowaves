package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/moby/go-archive"
	"github.com/moby/moby/api/types/build"
	mobyclient "github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/jsonmessage"
	"github.com/moby/patternmatcher/ignorefile"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const dockerignoreFileName = ".dockerignore"

// PullImage pulls the image of the given platform from the registry.
func PullImage(ctx context.Context, cl *mobyclient.Client, ref string, platform ocispec.Platform) error {
	resp, err := cl.ImagePull(ctx, ref, mobyclient.ImagePullOptions{Platforms: []ocispec.Platform{platform}})
	if err != nil {
		return fmt.Errorf("failed to pull image %q: %w", ref, err)
	}
	// Wait consumes the progress stream and returns the error reported in it, if any.
	if wErr := resp.Wait(ctx); wErr != nil {
		return errors.Join(fmt.Errorf("failed to pull image %q: %w", ref, wErr), resp.Close())
	}
	return resp.Close()
}

// BuildOptions holds parameters of an image build.
type BuildOptions struct {
	Name       string // Name is the tag of the built image.
	ContextDir string
	Dockerfile string // Dockerfile is the path to the Dockerfile relative to ContextDir.
	BuildArgs  map[string]*string
	Platform   ocispec.Platform
	Output     io.Writer // Output receives the build log.
	ErrOutput  io.Writer // ErrOutput receives the build error, if any.
}

// BuildImage builds the image with the classic builder. Files matched by .dockerignore in the context directory
// are excluded from the build context.
func BuildImage(ctx context.Context, cl *mobyclient.Client, opts BuildOptions) error {
	buildCtx, err := buildContext(opts.ContextDir, opts.Dockerfile)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer func() {
		_ = buildCtx.Close()
	}()
	resp, err := cl.ImageBuild(ctx, buildCtx, mobyclient.ImageBuildOptions{
		Tags:        []string{opts.Name},
		Dockerfile:  opts.Dockerfile,
		BuildArgs:   opts.BuildArgs,
		Platforms:   []ocispec.Platform{opts.Platform},
		Remove:      true,
		ForceRemove: true,
		Version:     build.BuilderV1,
	})
	if err != nil {
		return fmt.Errorf("failed to build image %q: %w", opts.Name, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	// Build errors are reported inside the response stream.
	if dErr := jsonmessage.DisplayJSONMessagesStream(resp.Body, opts.Output, 0, false, nil); dErr != nil {
		if opts.ErrOutput != nil {
			_, _ = fmt.Fprintln(opts.ErrOutput, dErr.Error())
		}
		return fmt.Errorf("failed to build image %q: %w", opts.Name, dErr)
	}
	return nil
}

// RemoveImagesByLabel removes all images with the given label, label should be in form "key=value".
func RemoveImagesByLabel(ctx context.Context, cl *mobyclient.Client, label string) error {
	images, err := cl.ImageList(ctx, mobyclient.ImageListOptions{
		Filters: make(mobyclient.Filters).Add("label", label),
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	for _, img := range images.Items {
		if _, rmErr := cl.ImageRemove(ctx, img.ID, mobyclient.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		}); rmErr != nil {
			return fmt.Errorf("failed to remove image %q: %w", img.ID, rmErr)
		}
	}
	return nil
}

// buildContext creates the tar archive of the context directory, skipping files matched by .dockerignore.
func buildContext(contextDir, dockerfile string) (io.ReadCloser, error) {
	var patterns []string
	f, err := os.Open(filepath.Clean(filepath.Join(contextDir, dockerignoreFileName)))
	switch {
	case errors.Is(err, os.ErrNotExist):
		// No .dockerignore, include everything.
	case err != nil:
		return nil, fmt.Errorf("failed to open %s: %w", dockerignoreFileName, err)
	default:
		patterns, err = ignorefile.ReadAll(f)
		if clErr := f.Close(); clErr != nil {
			err = errors.Join(err, clErr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", dockerignoreFileName, err)
		}
	}
	// Docker always sends these files to the daemon, even if they are excluded.
	patterns = append(patterns, "!"+filepath.ToSlash(dockerfile), "!"+dockerignoreFileName)
	return archive.TarWithOptions(contextDir, &archive.TarOptions{ExcludePatterns: patterns})
}
