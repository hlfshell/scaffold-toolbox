package mysql

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*Mysql)(nil)
var _ scaffold.NetworkAttachable = (*Mysql)(nil)
var _ scaffold.LabelAttachable = (*Mysql)(nil)
var _ scaffold.NamePrefixAttachable = (*Mysql)(nil)
var _ scaffold.EnvProvider = (*Mysql)(nil)
var _ scaffold.EndpointProvider = (*Mysql)(nil)
