package redis

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*Redis)(nil)
var _ scaffold.NetworkAttachable = (*Redis)(nil)
var _ scaffold.LabelAttachable = (*Redis)(nil)
var _ scaffold.NamePrefixAttachable = (*Redis)(nil)
var _ scaffold.EnvProvider = (*Redis)(nil)
var _ scaffold.EndpointProvider = (*Redis)(nil)
