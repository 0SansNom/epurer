package cleaner

import "testing"

func TestParseDockerReclaimable(t *testing.T) {
	output := `
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          5         2         1.5GB     800MB (53%)
Containers      3         1         20MB      0B
Local Volumes   2         0         100MB     100MB (100%)
Build Cache     10        0         2.5GB     1.2GB
`
	images, containers, volumes, buildCache := parseDockerReclaimable(output)
	if images != 800*1e6 {
		t.Errorf("images = %d, want 800MB", images)
	}
	if containers != 0 {
		t.Errorf("containers = %d, want 0", containers)
	}
	if volumes != 100*1e6 {
		t.Errorf("volumes = %d, want 100MB", volumes)
	}
	if buildCache != int64(1.2*1e9) {
		t.Errorf("build cache = %d, want 1.2GB", buildCache)
	}

	images, containers, volumes, buildCache = parseDockerReclaimable("Build Cache   not-a-size\n")
	if images != 0 || containers != 0 || volumes != 0 || buildCache != 0 {
		t.Errorf("unparsed = %d %d %d %d, want zeros", images, containers, volumes, buildCache)
	}
}
