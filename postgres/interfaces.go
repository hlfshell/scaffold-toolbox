package postgres

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*Postgres)(nil)
var _ scaffold.NetworkAttachable = (*Postgres)(nil)
var _ scaffold.LabelAttachable = (*Postgres)(nil)
var _ scaffold.NamePrefixAttachable = (*Postgres)(nil)
var _ scaffold.EnvProvider = (*Postgres)(nil)
var _ scaffold.EndpointProvider = (*Postgres)(nil)
