package cloud

func (s *ServiceTestSuite) TestDataCenterID() {
	s.Equal(s.service.scope.IonosMachine.Spec.DataCenterID, s.service.dataCenterID())
}

func (s *ServiceTestSuite) TestAPI() {
	s.Equal(s.service.scope.ClusterScope.IonosClient, s.service.api())
}
