package methods

// Valid: proper method ordering
type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Run() int {
	return s.compute()
}

func (s *Service) compute() int {
	return 1
}
