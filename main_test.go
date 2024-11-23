package main

import (
	"couldinaryuploader/cmd"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestUploadCommand(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []*cli.Command{cmd.UploadCommand}
	set := flag.NewFlagSet("test", 0)
	set.String("input", "testdata/mikrotikcdn.json", "doc")
	set.String("folder", "test-folder", "doc")
	set.String("output", "testdata/cdn", "doc")
	set.String("access", "CLOUDINARY_URL=cloudinary://test:test@test", "doc")

	ctx := cli.NewContext(app, set, nil)
	err := app.RunContext(ctx.Context, []string{"app", "upload"})

	assert.NoError(t, err)
}

func TestServeCommand(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []*cli.Command{cmd.ServeCommand}

	set := flag.NewFlagSet("test", 0)
	set.String("input", "testdata/cdn", "doc")

	ctx := cli.NewContext(app, set, nil)
	err := app.RunContext(ctx.Context, []string{"app", "serve"})

	assert.NoError(t, err)
}
