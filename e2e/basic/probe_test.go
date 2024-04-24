package basic

import (
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"testing"
)

func TestProbe(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
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
	err = web.AddFile("resources/html/index.html", "/usr/share/nginx/html/index.html", "0:0")
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
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		err = executor.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
		}

		err = web.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
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

	wget, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Equal(t, wget, "Hello World!\n")
}
