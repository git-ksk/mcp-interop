package client

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"os"
	"sort"
)

// inspectExecutableArchitectures reads only the executable image metadata. It
// never executes the client or reads client configuration/credential state.
// Script/wrapper launchers and unknown binary formats deliberately return no
// architecture evidence rather than inferring it from the host or runner.
func inspectExecutableArchitectures(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, nil
	}

	if fat, err := macho.OpenFat(path); err == nil {
		defer fat.Close()
		architectures := make([]string, 0, len(fat.Arches))
		for _, arch := range fat.Arches {
			if value := machoArchitecture(arch.File.Cpu); value != "" {
				architectures = appendUniqueArchitecture(architectures, value)
			}
		}
		sort.Strings(architectures)
		return architectures, nil
	}
	if file, err := macho.Open(path); err == nil {
		defer file.Close()
		if value := machoArchitecture(file.Cpu); value != "" {
			return []string{value}, nil
		}
		return nil, nil
	}
	if file, err := elf.Open(path); err == nil {
		defer file.Close()
		if value := elfArchitecture(file.Machine); value != "" {
			return []string{value}, nil
		}
		return nil, nil
	}
	if file, err := pe.Open(path); err == nil {
		defer file.Close()
		if value := peArchitecture(file.Machine); value != "" {
			return []string{value}, nil
		}
		return nil, nil
	}
	return nil, nil
}

func machoArchitecture(cpu macho.Cpu) string {
	switch cpu {
	case macho.Cpu386:
		return "386"
	case macho.CpuAmd64:
		return "amd64"
	case macho.CpuArm:
		return "arm"
	case macho.CpuArm64:
		return "arm64"
	default:
		return ""
	}
}

func elfArchitecture(machine elf.Machine) string {
	switch machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	default:
		return ""
	}
}

func peArchitecture(machine uint16) string {
	switch machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "amd64"
	case pe.IMAGE_FILE_MACHINE_ARM, pe.IMAGE_FILE_MACHINE_ARMNT:
		return "arm"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64"
	default:
		return ""
	}
}

func appendUniqueArchitecture(values []string, value string) []string {
	if containsArchitecture(values, value) {
		return values
	}
	return append(values, value)
}

func containsArchitecture(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
