package service

func (s *Service) ValidateCallback(params string) bool {
	return s.worldline.Validate(params)
}
