package commands_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry/bosh-bootloader/commands"
	"github.com/cloudfoundry/bosh-bootloader/fakes"
	"github.com/cloudfoundry/bosh-bootloader/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS Create LBs", func() {
	Describe("Execute", func() {
		var (
			command              commands.AWSCreateLBs
			terraformManager     *fakes.TerraformManager
			logger               *fakes.Logger
			cloudConfigManager   *fakes.CloudConfigManager
			stateStore           *fakes.StateStore
			environmentValidator *fakes.EnvironmentValidator
			incomingState        storage.State

			certPath  string
			keyPath   string
			chainPath string
		)

		BeforeEach(func() {
			terraformManager = &fakes.TerraformManager{}
			logger = &fakes.Logger{}
			cloudConfigManager = &fakes.CloudConfigManager{}
			stateStore = &fakes.StateStore{}
			environmentValidator = &fakes.EnvironmentValidator{}

			incomingState = storage.State{
				AWS: storage.AWS{
					AccessKeyID:     "some-access-key-id",
					SecretAccessKey: "some-secret-access-key",
					Region:          "some-region",
				},
				KeyPair: storage.KeyPair{
					Name: "some-key-pair",
				},
				TFState: "some-tf-state",
				BOSH: storage.BOSH{
					DirectorAddress:  "some-director-address",
					DirectorUsername: "some-director-username",
					DirectorPassword: "some-director-password",
				},
				EnvID: "some-env-id-timestamp",
			}

			tempCertFile, err := ioutil.TempFile("", "cert")
			Expect(err).NotTo(HaveOccurred())

			certificate := "some-cert"
			certPath = tempCertFile.Name()
			err = ioutil.WriteFile(certPath, []byte(certificate), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			tempKeyFile, err := ioutil.TempFile("", "key")
			Expect(err).NotTo(HaveOccurred())

			key := "some-key"
			keyPath = tempKeyFile.Name()
			err = ioutil.WriteFile(keyPath, []byte(key), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			tempChainFile, err := ioutil.TempFile("", "chain")
			Expect(err).NotTo(HaveOccurred())

			chain := "some-chain"
			chainPath = tempChainFile.Name()
			err = ioutil.WriteFile(chainPath, []byte(chain), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			command = commands.NewAWSCreateLBs(logger,
				cloudConfigManager,
				stateStore, terraformManager, environmentValidator)
		})

		Context("when lb type desired is cf", func() {
			var (
				statePassedToTerraform     storage.State
				stateReturnedFromTerraform storage.State
			)
			BeforeEach(func() {
				statePassedToTerraform = incomingState
				statePassedToTerraform.LB = storage.LB{
					Type: "cf",
					Cert: "some-cert",
					Key:  "some-key",
				}

				stateReturnedFromTerraform = statePassedToTerraform
				stateReturnedFromTerraform.TFState = "some-updated-tf-state"
				terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
			})

			It("creates a load balancer with certificate using terraform", func() {
				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "cf",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, incomingState)
				Expect(err).NotTo(HaveOccurred())

				Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
				Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
			})

			Context("when the optional chain is provided", func() {
				BeforeEach(func() {
					statePassedToTerraform.LB.Chain = "some-chain"

					stateReturnedFromTerraform = statePassedToTerraform
					stateReturnedFromTerraform.TFState = "some-updated-tf-state"
					terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
				})

				It("creates a load balancer with certificate using terraform", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:    "cf",
						CertPath:  certPath,
						KeyPath:   keyPath,
						ChainPath: chainPath,
					}, incomingState)
					Expect(err).NotTo(HaveOccurred())

					Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
					Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
				})
			})

			Context("when a domain is provided", func() {
				BeforeEach(func() {
					statePassedToTerraform.LB = storage.LB{
						Type:   "cf",
						Cert:   "some-cert",
						Key:    "some-key",
						Domain: "some-domain",
					}

					stateReturnedFromTerraform = statePassedToTerraform
					stateReturnedFromTerraform.TFState = "some-updated-tf-state"
					terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
				})

				It("creates dns records for provided domain", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "cf",
						CertPath: certPath,
						KeyPath:  keyPath,
						Domain:   "some-domain",
					}, incomingState)
					Expect(err).NotTo(HaveOccurred())

					Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
					Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
				})
			})

			Context("when a domain exists", func() {
				BeforeEach(func() {
					incomingState.LB = storage.LB{
						Type:   "cf",
						Cert:   "some-cert",
						Key:    "some-key",
						Domain: "some-domain",
					}
					statePassedToTerraform = incomingState

					stateReturnedFromTerraform = statePassedToTerraform
					stateReturnedFromTerraform.TFState = "some-updated-tf-state"
					terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
				})

				It("does not change domain", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "cf",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, incomingState)
					Expect(err).NotTo(HaveOccurred())

					Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
					Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
				})
			})

			Context("when lb type desired is concourse", func() {
				BeforeEach(func() {
					statePassedToTerraform = incomingState
					statePassedToTerraform.LB = storage.LB{
						Type: "concourse",
						Cert: "some-cert",
						Key:  "some-key",
					}

					stateReturnedFromTerraform = statePassedToTerraform
					stateReturnedFromTerraform.TFState = "some-updated-tf-state"
					terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
				})

				It("creates a load balancer with certificate using terraform", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, incomingState)
					Expect(err).NotTo(HaveOccurred())

					Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
					Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
				})

				Context("when optional chain is provided", func() {
					BeforeEach(func() {
						statePassedToTerraform.LB.Chain = "some-chain"

						stateReturnedFromTerraform = statePassedToTerraform
						stateReturnedFromTerraform.TFState = "some-updated-tf-state"
						terraformManager.ApplyCall.Returns.BBLState = stateReturnedFromTerraform
					})

					It("creates a load balancer with certificate using terraform", func() {
						err := command.Execute(commands.AWSCreateLBsConfig{
							LBType:    "concourse",
							CertPath:  certPath,
							KeyPath:   keyPath,
							ChainPath: chainPath,
						}, incomingState)
						Expect(err).NotTo(HaveOccurred())

						Expect(terraformManager.ApplyCall.Receives.BBLState).To(Equal(statePassedToTerraform))
						Expect(stateStore.SetCall.Receives[1].State).To(Equal(stateReturnedFromTerraform))
					})
				})
			})
		})

		Context("when the bbl environment has a BOSH director", func() {
			It("updates the cloud config with a state that has lb type", func() {
				terraformManager.ApplyCall.Returns.BBLState.LB.Type = "concourse"

				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "concourse",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, incomingState)
				Expect(err).NotTo(HaveOccurred())

				Expect(cloudConfigManager.UpdateCall.Receives.State.LB.Type).To(Equal("concourse"))
			})
		})

		Context("when the bbl environment does not have a BOSH director", func() {
			BeforeEach(func() {
				incomingState = storage.State{
					NoDirector: true,
					Stack: storage.Stack{
						Name:   "some-stack",
						BOSHAZ: "some-bosh-az",
					},
					AWS: storage.AWS{
						AccessKeyID:     "some-access-key-id",
						SecretAccessKey: "some-secret-access-key",
						Region:          "some-region",
					},
					KeyPair: storage.KeyPair{
						Name: "some-key-pair",
					},
					EnvID: "some-env-id-timestamp",
				}
			})

			It("does not call cloudConfigManager", func() {
				terraformManager.ApplyCall.Returns.BBLState = incomingState

				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "concourse",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, incomingState)
				Expect(err).NotTo(HaveOccurred())

				Expect(cloudConfigManager.UpdateCall.CallCount).To(Equal(0))
			})
		})

		Context("when --skip-if-exists is provided", func() {
			It("no-ops when lb exists", func() {
				incomingState.Stack.LBType = "cf"
				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:       "concourse",
					CertPath:     certPath,
					KeyPath:      keyPath,
					SkipIfExists: true,
				}, incomingState)
				Expect(err).NotTo(HaveOccurred())

				Expect(terraformManager.ApplyCall.CallCount).To(Equal(0))

				Expect(logger.PrintlnCall.Receives.Message).To(Equal(`lb type "cf" exists, skipping...`))
			})

			DescribeTable("creates the lb if the lb does not exist",
				func(currentLBType string) {
					incomingState.Stack.LBType = currentLBType
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:       "concourse",
						CertPath:     certPath,
						KeyPath:      keyPath,
						SkipIfExists: true,
					}, incomingState)
					Expect(err).NotTo(HaveOccurred())

					Expect(terraformManager.ApplyCall.CallCount).To(Equal(1))
				},
				Entry("when the current lb-type is 'none'", "none"),
				Entry("when the current lb-type is ''", ""),
			)
		})

		Context("invalid lb type", func() {
			It("returns an error", func() {
				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "some-invalid-lb",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, incomingState)
				Expect(err).To(MatchError("\"some-invalid-lb\" is not a valid lb type, valid lb types are: concourse and cf"))
			})

			It("returns a helpful error when no lb type is provided", func() {
				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, incomingState)
				Expect(err).To(MatchError("--type is a required flag"))
			})
		})

		It("returns an error when the environment validator fails", func() {
			environmentValidator.ValidateCall.Returns.Error = errors.New("environment not found")

			err := command.Execute(commands.AWSCreateLBsConfig{
				LBType:   "concourse",
				CertPath: certPath,
				KeyPath:  keyPath,
			}, incomingState)

			Expect(environmentValidator.ValidateCall.Receives.State).To(Equal(incomingState))
			Expect(err).To(MatchError("environment not found"))
		})

		Context("state manipulation", func() {
			Context("when the env id does not exist", func() {
				It("saves state with new certificate name and lb type", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, storage.State{})
					Expect(err).NotTo(HaveOccurred())

					Expect(stateStore.SetCall.CallCount).To(Equal(2))
					state := stateStore.SetCall.Receives[0].State
					Expect(state.LB.Type).To(Equal("concourse"))
				})
			})
		})

		Context("failure cases", func() {
			DescribeTable("returns an error when an lb already exists",
				func(newLbType, oldLbType string) {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, storage.State{
						Stack: storage.Stack{
							LBType: oldLbType,
						},
					})
					Expect(err).To(MatchError(fmt.Sprintf("bbl already has a %s load balancer attached, please remove the previous load balancer before attaching a new one", oldLbType)))
				},
				Entry("when the previous lb type is concourse", "concourse", "cf"),
				Entry("when the previous lb type is cf", "cf", "concourse"),
			)

			Context("when lb is cf and cert path is invalid", func() {
				It("returns an error", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "cf",
						CertPath: "/fake/cert/path",
						KeyPath:  keyPath,
					}, storage.State{
						TFState: "some-tf-state",
					})
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			Context("when lb is cf and key path is invalid", func() {
				var certPath string

				BeforeEach(func() {
					tempCertFile, err := ioutil.TempFile("", "cert")
					Expect(err).NotTo(HaveOccurred())
					certPath = tempCertFile.Name()
				})

				It("returns an error", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "cf",
						CertPath: certPath,
						KeyPath:  "/fake/key/path",
					}, storage.State{
						TFState: "some-tf-state",
					})
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			Context("when lb is cf and chain path is invalid", func() {
				var (
					certPath string
					keyPath  string
				)

				BeforeEach(func() {
					tempCertFile, err := ioutil.TempFile("", "cert")
					Expect(err).NotTo(HaveOccurred())
					certPath = tempCertFile.Name()

					tempKeyFile, err := ioutil.TempFile("", "key")
					Expect(err).NotTo(HaveOccurred())
					keyPath = tempKeyFile.Name()
				})

				It("returns an error", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:    "cf",
						CertPath:  certPath,
						KeyPath:   keyPath,
						ChainPath: "/fake/chain/path",
					}, storage.State{
						TFState: "some-tf-state",
					})
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			Context("when terraform manager fails to apply", func() {
				var (
					certPath string
					keyPath  string
				)

				BeforeEach(func() {
					tempCertFile, err := ioutil.TempFile("", "cert")
					Expect(err).NotTo(HaveOccurred())
					certPath = tempCertFile.Name()

					tempKeyFile, err := ioutil.TempFile("", "key")
					Expect(err).NotTo(HaveOccurred())
					keyPath = tempKeyFile.Name()
				})

				It("returns an error", func() {
					terraformManager.ApplyCall.Returns.Error = errors.New("failed to apply")

					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, storage.State{
						TFState: "some-tf-state",
					})
					Expect(err).To(MatchError("failed to apply"))
				})
			})

			Context("when the terraform manager fails with terraformManagerError", func() {
				var (
					certPath string
					keyPath  string

					managerError *fakes.TerraformManagerError
				)

				BeforeEach(func() {
					tempCertFile, err := ioutil.TempFile("", "cert")
					Expect(err).NotTo(HaveOccurred())
					certPath = tempCertFile.Name()

					tempKeyFile, err := ioutil.TempFile("", "key")
					Expect(err).NotTo(HaveOccurred())
					keyPath = tempKeyFile.Name()

					managerError = &fakes.TerraformManagerError{}
					managerError.BBLStateCall.Returns.BBLState = storage.State{
						TFState: "some-partial-tf-state",
					}
					managerError.ErrorCall.Returns = "cannot apply"
					terraformManager.ApplyCall.Returns.Error = managerError
				})

				It("saves the bbl state and returns the error", func() {
					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, storage.State{
						TFState: "some-tf-state",
					})
					Expect(err).To(MatchError("cannot apply"))

					Expect(stateStore.SetCall.CallCount).To(Equal(2))
					Expect(stateStore.SetCall.Receives[1].State).To(Equal(storage.State{
						TFState: "some-partial-tf-state",
					}))
				})

				Context("when the terraform manager error fails to return a bbl state", func() {
					BeforeEach(func() {
						managerError.BBLStateCall.Returns.Error = errors.New("failed to retrieve bbl state")
					})

					It("saves the bbl state and returns the error", func() {
						err := command.Execute(commands.AWSCreateLBsConfig{
							LBType:   "concourse",
							CertPath: certPath,
							KeyPath:  keyPath,
						}, storage.State{
							TFState: "some-tf-state",
						})
						Expect(err).To(MatchError("the following errors occurred:\ncannot apply,\nfailed to retrieve bbl state"))
					})
				})

				Context("when we fail to set the bbl state", func() {
					BeforeEach(func() {
						managerError.BBLStateCall.Returns.BBLState = storage.State{
							TFState: "some-partial-tf-state",
						}

						stateStore.SetCall.Returns = []fakes.SetCallReturn{
							{},
							{errors.New("failed to set bbl state")},
						}
					})

					It("attempts to save the bbl state and returns the error", func() {
						err := command.Execute(commands.AWSCreateLBsConfig{
							LBType:   "concourse",
							CertPath: certPath,
							KeyPath:  keyPath,
						}, storage.State{
							TFState: "some-tf-state",
						})

						Expect(err).To(MatchError("the following errors occurred:\ncannot apply,\nfailed to set bbl state"))
					})
				})
			})

			Context("when cloud config manager update fails", func() {
				It("returns an error", func() {
					cloudConfigManager.UpdateCall.Returns.Error = errors.New("failed to update cloud config")

					err := command.Execute(commands.AWSCreateLBsConfig{
						LBType:   "concourse",
						CertPath: certPath,
						KeyPath:  keyPath,
					}, storage.State{})
					Expect(err).To(MatchError("failed to update cloud config"))
				})
			})

			It("returns an error when the state fails to save", func() {
				stateStore.SetCall.Returns = []fakes.SetCallReturn{{errors.New("failed to save state")}}
				err := command.Execute(commands.AWSCreateLBsConfig{
					LBType:   "concourse",
					CertPath: certPath,
					KeyPath:  keyPath,
				}, storage.State{})
				Expect(err).To(MatchError("failed to save state"))
			})
		})
	})
})
