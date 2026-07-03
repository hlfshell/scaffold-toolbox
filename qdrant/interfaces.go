package qdrant

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*Qdrant)(nil)
var _ scaffold.NetworkAttachable = (*Qdrant)(nil)
var _ scaffold.LabelAttachable = (*Qdrant)(nil)
var _ scaffold.NamePrefixAttachable = (*Qdrant)(nil)
var _ scaffold.EnvProvider = (*Qdrant)(nil)
var _ scaffold.EndpointProvider = (*Qdrant)(nil)
