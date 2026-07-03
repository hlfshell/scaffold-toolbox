package localstack

import "github.com/hlfshell/scaffold"

var _ scaffold.Service = (*LocalStack)(nil)
var _ scaffold.NetworkAttachable = (*LocalStack)(nil)
var _ scaffold.LabelAttachable = (*LocalStack)(nil)
var _ scaffold.NamePrefixAttachable = (*LocalStack)(nil)
var _ scaffold.EnvProvider = (*LocalStack)(nil)
var _ scaffold.EndpointProvider = (*LocalStack)(nil)
