package vinfo

import "github.com/wavesplatform/gowaves/pkg/ride/ast"

type ConstructorStructInfo struct {
	RideName   string
	GoName     string
	ArgsNumber int
}

type VersionInfo struct {
	Version        ast.LibraryVersion
	NewStructs     []ConstructorStructInfo // new structs or modified structs
	RemovedStructs []string                // structs removed in this version
}

func NewVersionInfo(version ast.LibraryVersion) *VersionInfo {
	return &VersionInfo{
		Version:        version,
		NewStructs:     make([]ConstructorStructInfo, 0),
		RemovedStructs: make([]string, 0),
	}
}

type VersionInfos map[ast.LibraryVersion]*VersionInfo

func (vInfos VersionInfos) AddNewStruct(version ast.LibraryVersion, info ConstructorStructInfo) {
	if _, ok := vInfos[version]; !ok {
		vInfos[version] = NewVersionInfo(version)
	}

	vInfo := vInfos[version]
	vInfo.NewStructs = append(vInfo.NewStructs, info)
}

func (vInfos VersionInfos) AddRemoved(version ast.LibraryVersion, name string) {
	if _, ok := vInfos[version]; !ok {
		vInfos[version] = NewVersionInfo(version)
	}

	vInfo := vInfos[version]
	vInfo.RemovedStructs = append(vInfo.RemovedStructs, name)
}

var globalVerInfos = make(VersionInfos)

func GetVerInfos() VersionInfos {
	return globalVerInfos
}
