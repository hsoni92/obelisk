package main

import "github.com/hsoni92/obelisk/internal/release"

// Set at release time: -ldflags "-X main.buildNumber=N -X main.buildCommit=SHA".
// Empty for developer builds.
var (
	buildNumber string
	buildCommit string
)

const (
	defaultRepo = release.DefaultRepo

	exitUpdateAvailable = 10 // `update --check`: a newer build exists
	exitCannotResolve   = 11 // network or GitHub error; says nothing about being up to date
)

var errNoRelease = release.ErrNoRelease

func parseBuildTag(tag string) (int, bool) { return release.ParseBuildTag(tag) }
func releaseBaseURL() string               { return release.BaseURL() }
func releaseRepo(metaRepo string) string   { return release.Repo(metaRepo) }

func resolveRelease(base, repo, tag string) (int, error) { return release.Resolve(base, repo, tag) }
