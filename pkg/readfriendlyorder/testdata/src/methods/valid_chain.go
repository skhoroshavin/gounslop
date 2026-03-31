package methods

// Valid: method dependency chain - bootstrap calls compute, run calls compute
type Processor struct{}

func NewProcessor() *Processor { return &Processor{} }

func (p *Processor) Bootstrap() int {
	return p.compute()
}

func (p *Processor) Run() int {
	return p.compute()
}

func (p *Processor) compute() int {
	return 1
}
