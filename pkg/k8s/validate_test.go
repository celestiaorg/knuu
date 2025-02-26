package k8s

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stretchr/testify/assert"
)

func TestValidateDNS1123Label(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid Label", "valid-label", nil},
		{"Invalid Label", "Invalid_Label!", ErrInvalidNamespaceName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDNS1123Label(test.input, ErrInvalidNamespaceName)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateDNS1123Subdomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid Subdomain", "valid-subdomain", nil},
		{"Invalid Subdomain", "Invalid_Subdomain!", ErrInvalidConfigMapName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDNS1123Subdomain(test.input, ErrInvalidConfigMapName)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid Namespace", "valid-namespace", nil},
		{"Invalid Namespace", "Invalid_Namespace!", ErrInvalidNamespaceName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateNamespace(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected error
	}{
		{"Valid Labels", map[string]string{"key": "value"}, nil},
		{"Invalid Label Key", map[string]string{"Invalid Key!": "value"}, ErrInvalidLabelKey},
		{"Invalid Label Value", map[string]string{"key": "Invalid Value!"}, ErrInvalidLabelValue},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLabels(test.labels)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidatePorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []int
		expected error
	}{
		{"Valid Ports", []int{80, 443}, nil},
		{"Invalid Port", []int{0, 80}, ErrInvalidPort},
		{"Invalid Port", []int{80, 70000}, ErrInvalidPort},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePorts(test.ports)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid Container Name", "valid-name", nil},
		{"Invalid Container Name", "Invalid-Name!", ErrInvalidContainerName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateContainerName(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidatePodConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    PodConfig
		expected error
	}{
		{"Valid Pod Config", PodConfig{
			Name:        "valid-name",
			Namespace:   "valid-namespace",
			Labels:      map[string]string{"key": "value"},
			Annotations: map[string]string{"key": "value"},
			ContainerConfig: ContainerConfig{
				Name:  "container",
				Image: "image",
				Volumes: []*Volume{
					{Path: "/data", Size: resource.MustParse("1Gi")},
				},
				Files: []*File{
					{Source: "source", Dest: "dest"},
				},
			},
		}, nil},
		{"Invalid Pod Config", PodConfig{
			Name:        "Invalid-Name!",
			Namespace:   "valid-namespace",
			Labels:      map[string]string{"key": "value"},
			Annotations: map[string]string{"key": "value"},
			ContainerConfig: ContainerConfig{
				Name:  "container",
				Image: "image",
				Volumes: []*Volume{
					{Path: "/data", Size: resource.MustParse("1Gi")},
				},
				Files: []*File{
					{Source: "source", Dest: "dest"},
				},
			},
		}, ErrInvalidPodName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePodConfig(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateGroupVersionResource(t *testing.T) {
	tests := []struct {
		name     string
		input    schema.GroupVersionResource
		expected error
	}{
		{"Valid GVR", schema.GroupVersionResource{Group: "group", Version: "v1", Resource: "resource"}, nil},
		{"Invalid GVR", schema.GroupVersionResource{Group: "", Version: "v1", Resource: "resource"}, ErrInvalidGroupVersionResource},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateGroupVersionResource(&test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid Role Name", "valid-role", nil},
		{"Invalid Role Name", "Invalid_Role!", ErrInvalidRoleName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRoleName(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidatePVCName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected error
	}{
		{"Valid PVC Name", "valid-pvc", nil},
		{"Invalid PVC Name", "Invalid_PVC!", ErrInvalidPVCName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePVCName(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestValidatePVCSize(t *testing.T) {
	tests := []struct {
		name     string
		input    resource.Quantity
		expected error
	}{
		{"Valid PVC Size", resource.MustParse("1Gi"), nil},
		{"Zero PVC Size", resource.MustParse("0Gi"), ErrPVCSizeZero},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePVCSize(test.input)
			assert.Equal(t, test.expected, err)
		})
	}
}
