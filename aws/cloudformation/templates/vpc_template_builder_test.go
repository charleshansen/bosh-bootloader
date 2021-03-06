package templates_test

import (
	"fmt"
	"math/rand"

	"github.com/cloudfoundry/bosh-bootloader/aws/cloudformation/templates"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VPCTemplateBuilder", func() {
	var builder templates.VPCTemplateBuilder

	BeforeEach(func() {
		builder = templates.NewVPCTemplateBuilder()
	})

	Describe("VPC", func() {
		It("returns a template with the VPC-related parameters", func() {
			vpc := builder.VPC("")

			Expect(vpc.Parameters).To(HaveLen(1))
			Expect(vpc.Parameters).To(HaveKeyWithValue("VPCCIDR", templates.Parameter{
				Description: "CIDR block for the VPC.",
				Type:        "String",
				Default:     "10.0.0.0/16",
			}))
		})

		It("returns a template with the VPC-related resources", func() {
			envID := fmt.Sprintf("some-env-id-%v", rand.Int())
			vpc := builder.VPC(envID)

			Expect(vpc.Resources).To(HaveLen(3))
			Expect(vpc.Resources).To(HaveKeyWithValue("VPC", templates.Resource{
				Type: "AWS::EC2::VPC",
				Properties: templates.VPC{
					CidrBlock: templates.Ref{"VPCCIDR"},
					Tags: []templates.Tag{
						{
							Value: fmt.Sprintf("vpc-%s", envID),
							Key:   "Name",
						},
					},
				},
				DeletionPolicy: "Retain",
			}))

			Expect(vpc.Resources).To(HaveKeyWithValue("VPCGatewayInternetGateway", templates.Resource{
				Type:           "AWS::EC2::InternetGateway",
				DeletionPolicy: "Retain",
			}))

			Expect(vpc.Resources).To(HaveKeyWithValue("VPCGatewayAttachment", templates.Resource{
				Type: "AWS::EC2::VPCGatewayAttachment",
				Properties: templates.VPCGatewayAttachment{
					VpcId:             templates.Ref{"VPC"},
					InternetGatewayId: templates.Ref{"VPCGatewayInternetGateway"},
				},
				DeletionPolicy: "Retain",
			}))
		})

		It("returns a template with the VPC-related outputs", func() {
			vpc := builder.VPC("")

			Expect(vpc.Outputs).To(HaveKeyWithValue("VPCID", templates.Output{
				Value: templates.Ref{Ref: "VPC"},
			}))

			Expect(vpc.Outputs).To(HaveKeyWithValue("VPCInternetGatewayID", templates.Output{
				Value: templates.Ref{Ref: "VPCGatewayInternetGateway"},
			}))
		})
	})
})
