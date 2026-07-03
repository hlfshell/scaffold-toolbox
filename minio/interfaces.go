package minio

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*MinIO)(nil)
var _ scaffold.NetworkAttachable = (*MinIO)(nil)
var _ scaffold.LabelAttachable = (*MinIO)(nil)
var _ scaffold.NamePrefixAttachable = (*MinIO)(nil)
var _ scaffold.EnvProvider = (*MinIO)(nil)
var _ scaffold.EndpointProvider = (*MinIO)(nil)
