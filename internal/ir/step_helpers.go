package ir

// WithName attaches a declared step name to a step value. Used by the parser
// to label steps post-decode so logs and diagnostics can refer to them.

func (s ValidateStep) WithName(n string) ValidateStep         { s.Name = n; return s }
func (s LoadStep) WithName(n string) LoadStep                 { s.Name = n; return s }
func (s AuthorizeStep) WithName(n string) AuthorizeStep       { s.Name = n; return s }
func (s ExternalCallStep) WithName(n string) ExternalCallStep { s.Name = n; return s }
func (s UpdateStep) WithName(n string) UpdateStep             { s.Name = n; return s }
func (s CreateStep) WithName(n string) CreateStep             { s.Name = n; return s }
func (s DeleteStep) WithName(n string) DeleteStep             { s.Name = n; return s }
func (s QueryStep) WithName(n string) QueryStep               { s.Name = n; return s }
func (s EmitEventStep) WithName(n string) EmitEventStep       { s.Name = n; return s }
func (s ConsumeStep) WithName(n string) ConsumeStep           { s.Name = n; return s }
func (s IfStep) WithName(n string) IfStep                     { s.Name = n; return s }
func (s ParallelStep) WithName(n string) ParallelStep         { s.Name = n; return s }
func (s SagaStep) WithName(n string) SagaStep                 { s.Name = n; return s }
func (s CompensateStep) WithName(n string) CompensateStep     { s.Name = n; return s }
func (s RetryStep) WithName(n string) RetryStep               { s.Name = n; return s }
func (s CacheStep) WithName(n string) CacheStep               { s.Name = n; return s }
func (s LogStep) WithName(n string) LogStep                   { s.Name = n; return s }
func (s PolicyStep) WithName(n string) PolicyStep             { s.Name = n; return s }
func (s TransactionStep) WithName(n string) TransactionStep   { s.Name = n; return s }
func (s ReturnStep) WithName(n string) ReturnStep             { s.Name = n; return s }
