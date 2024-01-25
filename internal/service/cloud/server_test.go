package cloud

import (
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
	"github.com/stretchr/testify/suite"

	clienttest "github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud/clienttest"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/ptr"
)

type serverSuite struct {
	ServiceTestSuite
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(serverSuite))
}

func (s *serverSuite) TestGetServer() {
	s.mockListSevers().Return(&sdk.Servers{Items: &[]sdk.Server{}}, nil)
}

//nolint:unused
func (s *serverSuite) exampleServer() sdk.Server {
	return sdk.Server{
		Id: ptr.To("1"),
		Metadata: &sdk.DatacenterElementMetadata{
			State: ptr.To(sdk.Available),
		},
		Properties: &sdk.ServerProperties{
			AvailabilityZone: ptr.To("AUTO"),
			BootVolume: &sdk.ResourceReference{
				Id:   ptr.To("1"),
				Type: ptr.To(sdk.VOLUME),
			},
			Name:    nil,
			VmState: nil,
		},
	}
}

func (s *serverSuite) mockListSevers() *clienttest.MockClient_ListServers_Call {
	return s.ionosClient.EXPECT().ListServers(s.ctx, s.service.datacenterID())
}
