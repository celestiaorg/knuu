package basic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestProbe(t *testing.T) {
	t.Parallel()
	// Setup

	// Ideally this has to be defined in the test suit setup
	exe := e2e.Executor{
		Kn: knuu.GetKnuuObj(),
	}

	ctx := context.Background()
	executor, err := exe.NewInstance(ctx, "prob-executor")
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	web, err := knuu.NewInstance("web")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = web.SetImage("docker.io/nginx:latest")
	if err != nil {
		t.Fatalf("Error setting image '%v':", err)
	}
	web.AddPortTCP(80)
	_, err = web.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}
	err = web.AddVolumeWithOwner("/usr/share/nginx/html", "1Gi", 0)
	if err != nil {
		t.Fatalf("Error adding volume: %v", err)
	}
	err = web.AddFile("../system/resources/html/index.html", "/usr/share/nginx/html/index.html", "0:0")
	if err != nil {
		t.Fatalf("Error adding file '%v':", err)
	}
	livenessProbe := v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: "/",
				Port: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
			},
		},
		InitialDelaySeconds: 10,
	}
	err = web.SetLivenessProbe(&livenessProbe)
	if err != nil {
		t.Fatalf("Error setting readiness probe '%v':", err)
	}
	err = web.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
	}

	t.Cleanup(func() {
		// after refactor, we can use instance.BatchDestroy for simplicity
		err := executor.Execution().Destroy(ctx)
		if err != nil {
			t.Logf("Error destroying instance: %v", err)
		}

		err = web.Destroy()
		if err != nil {
			t.Logf("Error destroying instance: %v", err)
		}
	})

	// Test logic

	webIP, err := web.GetIP()
	if err != nil {
		t.Fatalf("Error getting IP '%v':", err)
	}

	err = web.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = web.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Contains(t, wget, "Hello World!")
}
