// Package skills embeds the skill directories of this repository and
// provides APIs to enumerate and install them for AI coding tools.
package skills

import (
	"embed"
	"io/fs"
)

// Skill directories at the repository root. A new skill must be added
// to this list to get embedded into the binary.
//
//go:embed android-camera-dev android-device-debugging arch-dev-workflow async-job-sse-upload-dev branching-workflow docker-edge-ops-pitfalls edge-offline-pipeline-dev frontend-dev-workflow git-identity installer-bundle-design installer-preflight installer-target-execution jenkins-pipeline-dev mqtt-device-gateway-dev ohos-media-dev onvif-rtsp-dev open-source-contribution redis-stream-rpc robot-cameras skill-distillation skill-refresh subagent-parallel-dev ui-responsive-list user-facing-errors windows-repo-tooling-pitfalls
var fsys embed.FS

// FS returns the read-only filesystem with all embedded skills.
func FS() fs.FS { return fsys }
