package engine

import "strings"

func isVMLayer(layer string) bool {
	switch layer {
	case "vm-block", "vm-file", "vm-object", "vm-application", "vm-afa":
		return true
	default:
		return false
	}
}

func isVMSSH(in RunInput) bool {
	return isVMLayer(in.Profile.Layer) && strings.Contains(in.Target, "@")
}
