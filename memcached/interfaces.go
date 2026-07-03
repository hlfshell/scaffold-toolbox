package memcached

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*Memcached)(nil)
var _ scaffold.NetworkAttachable = (*Memcached)(nil)
var _ scaffold.LabelAttachable = (*Memcached)(nil)
var _ scaffold.NamePrefixAttachable = (*Memcached)(nil)
var _ scaffold.EnvProvider = (*Memcached)(nil)
var _ scaffold.EndpointProvider = (*Memcached)(nil)
