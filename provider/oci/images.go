package oci

type InstanceType string

const (
	BareMetal      InstanceType = "metal"
	VirtualMachine InstanceType = "virtual"
)

// ShapeSpec holds information about a shapes resource allocation
type ShapeSpec struct {
	// Cpus is the number of CPU cores available to the instance
	Cpus int
	// Memory is the amount of RAM available to the instance
	Memory int
	Type   InstanceType
}

var shapeSpecs = map[string]ShapeSpec{
	"VM.Standard1.1": ShapeSpec{
		Cpus:   1,
		Memory: 7168,
		Type:   VirtualMachine,
	},
	"VM.Standard2.1": ShapeSpec{
		Cpus:   1,
		Memory: 15360,
		Type:   VirtualMachine,
	},
	"VM.Standard1.2": ShapeSpec{
		Cpus:   2,
		Memory: 14336,
		Type:   VirtualMachine,
	},
	"VM.Standard2.2": ShapeSpec{
		Cpus:   2,
		Memory: 30720,
		Type:   VirtualMachine,
	},
	"VM.Standard1.4": ShapeSpec{
		Cpus:   4,
		Memory: 28672,
		Type:   VirtualMachine,
	},
	"VM.Standard2.4": ShapeSpec{
		Cpus:   4,
		Memory: 61440,
		Type:   VirtualMachine,
	},
	"VM.Standard1.8": ShapeSpec{
		Cpus:   8,
		Memory: 57344,
		Type:   VirtualMachine,
	},
	"VM.Standard2.8": ShapeSpec{
		Cpus:   8,
		Memory: 122880,
		Type:   VirtualMachine,
	},
	"VM.Standard1.16": ShapeSpec{
		Cpus:   16,
		Memory: 114688,
		Type:   VirtualMachine,
	},
	"VM.Standard2.16": ShapeSpec{
		Cpus:   16,
		Memory: 245760,
		Type:   VirtualMachine,
	},
	"VM.Standard2.24": ShapeSpec{
		Cpus:   24,
		Memory: 327680,
		Type:   VirtualMachine,
	},
	"VM.DenseIO1.4": ShapeSpec{
		Cpus:   4,
		Memory: 61440,
		Type:   VirtualMachine,
	},
	"VM.DenseIO1.8": ShapeSpec{
		Cpus:   8,
		Memory: 122880,
		Type:   VirtualMachine,
	},
	"VM.DenseIO2.8": ShapeSpec{
		Cpus:   8,
		Memory: 122880,
		Type:   VirtualMachine,
	},
	"VM.DenseIO1.16": ShapeSpec{
		Cpus:   16,
		Memory: 245760,
		Type:   VirtualMachine,
	},

	"VM.DenseIO2.16": ShapeSpec{
		Cpus:   16,
		Memory: 245760,
		Type:   VirtualMachine,
	},
	"VM.DenseIO2.24": ShapeSpec{
		Cpus:   24,
		Memory: 327680,
		Type:   VirtualMachine,
	},
	"BM.Standard1.36": ShapeSpec{
		Cpus:   36,
		Memory: 262144,
		Type:   BareMetal,
	},
	"BM.Standard2.52": ShapeSpec{
		Cpus:   52,
		Memory: 786432,
		Type:   BareMetal,
	},
	"BM.HighIO1.36": ShapeSpec{
		Cpus:   36,
		Memory: 524288,
		Type:   BareMetal,
	},
	"BM.DenseIO1.36": ShapeSpec{
		Cpus:   1,
		Memory: 7168,
		Type:   BareMetal,
	},
	"BM.DenseIO2.52": ShapeSpec{
		Cpus:   52,
		Memory: 786432,
		Type:   BareMetal,
	},
}
