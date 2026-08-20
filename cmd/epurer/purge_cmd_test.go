package main

import (
	"testing"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/purge"
)

func testProjectFor(name string, artifacts ...purge.Artifact) purge.Project {
	return purge.Project{Root: "/tmp/" + name, Name: name, Type: "Node.js", Artifacts: artifacts}
}

func testArtifactFor(path string, size int64, selected bool) purge.Artifact {
	return purge.Artifact{
		CleanTarget: cleaner.CleanTarget{Path: path, SizeBytes: size, Safety: config.Moderate},
		Kind:        "node_modules",
		Selected:    selected,
	}
}

func TestCandidateArtifacts_OnlyPreselectedByDefault(t *testing.T) {
	orig := purgeAll
	defer func() { purgeAll = orig }()
	purgeAll = false

	projects := []purge.Project{
		testProjectFor("old", testArtifactFor("/tmp/old/node_modules", 10, true)),
		testProjectFor("new", testArtifactFor("/tmp/new/node_modules", 20, false)),
	}

	got := candidateArtifacts(projects)
	if len(got) != 1 || got[0].Path != "/tmp/old/node_modules" {
		t.Errorf("candidateArtifacts() = %+v, want only the preselected artifact", got)
	}
}

func TestCandidateArtifacts_AllIgnoresPreselection(t *testing.T) {
	orig := purgeAll
	defer func() { purgeAll = orig }()
	purgeAll = true

	projects := []purge.Project{
		testProjectFor("old", testArtifactFor("/tmp/old/node_modules", 10, true)),
		testProjectFor("new", testArtifactFor("/tmp/new/node_modules", 20, false)),
	}

	got := candidateArtifacts(projects)
	if len(got) != 2 {
		t.Errorf("candidateArtifacts() with --all = %d artifacts, want 2", len(got))
	}
}

func TestCandidateArtifacts_EmptyProjects(t *testing.T) {
	orig := purgeAll
	defer func() { purgeAll = orig }()
	purgeAll = false

	if got := candidateArtifacts(nil); len(got) != 0 {
		t.Errorf("candidateArtifacts(nil) = %+v, want empty", got)
	}
}
