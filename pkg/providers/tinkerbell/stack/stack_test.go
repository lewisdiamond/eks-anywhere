package stack_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/features"
	filewritermocks "github.com/aws/eks-anywhere/pkg/filewriter/mocks"
	"github.com/aws/eks-anywhere/pkg/providers/tinkerbell/stack"
	"github.com/aws/eks-anywhere/pkg/providers/tinkerbell/stack/mocks"
	"github.com/aws/eks-anywhere/pkg/types"
	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	overridesFileName = "tinkerbell-chart-overrides.yaml"
	boots             = "boots"
	testIP            = "1.2.3.4"

	// TODO: remove this once the chart is added to bundle
	helmChartOci     = "oci://public.ecr.aws/h6q6q4n4/tinkerbell"
	helmChartName    = "tinkerbell"
	helmChartVersion = "0.1.0"
)

func getTinkBundle() releasev1alpha1.TinkerbellStackBundle {
	return releasev1alpha1.TinkerbellStackBundle{
		Tink: releasev1alpha1.TinkBundle{
			TinkWorker: releasev1alpha1.Image{URI: "tink-worker:latest"},
		},
		Boots: releasev1alpha1.TinkerbellServiceBundle{
			Image: releasev1alpha1.Image{URI: "boots:latest"},
		},
		Hegel: releasev1alpha1.TinkerbellServiceBundle{
			Image: releasev1alpha1.Image{URI: "hegel:latest"},
		},
	}
}

func TestTinkerbellStackInstallWithAllOptionsSuccess(t *testing.T) {
	t.Setenv(features.TinkerbellProviderEnvVar, "true")
	mockCtrl := gomock.NewController(t)
	docker := mocks.NewMockDocker(mockCtrl)
	helm := mocks.NewMockHelm(mockCtrl)
	writer := filewritermocks.NewMockFileWriter(mockCtrl)
	cluster := &types.Cluster{Name: "test"}
	ctx := context.Background()

	s := stack.NewInstaller(docker, writer, helm)

	writer.EXPECT().Write(overridesFileName, gomock.Any()).Return(overridesFileName, nil)

	helm.EXPECT().InstallChartWithValuesFile(ctx, helmChartName, helmChartOci, helmChartVersion, cluster.KubeconfigFile, overridesFileName)

	if err := s.Install(ctx,
		getTinkBundle(),
		testIP,
		cluster.KubeconfigFile,
		stack.WithNamespace(constants.EksaSystemNamespace, true),
		stack.WithBootsOnKubernetes(),
	); err != nil {
		t.Fatalf("failed to install Tinkerbell stack: %v", err)
	}
}

func TestTinkerbellStackInstallWithBootsOnDockerSuccess(t *testing.T) {
	t.Setenv(features.TinkerbellProviderEnvVar, "true")
	mockCtrl := gomock.NewController(t)
	docker := mocks.NewMockDocker(mockCtrl)
	helm := mocks.NewMockHelm(mockCtrl)
	writer := filewritermocks.NewMockFileWriter(mockCtrl)
	cluster := &types.Cluster{Name: "test"}
	ctx := context.Background()
	s := stack.NewInstaller(docker, writer, helm)

	writer.EXPECT().Write(overridesFileName, gomock.Any()).Return(overridesFileName, nil)
	helm.EXPECT().InstallChartWithValuesFile(ctx, helmChartName, helmChartOci, helmChartVersion, cluster.KubeconfigFile, overridesFileName)
	docker.EXPECT().Run(ctx, "boots:latest",
		boots,
		[]string{"-kubeconfig", "/kubeconfig", "-dhcp-addr", "0.0.0.0:67"},
		"-v", gomock.Any(),
		"--network", "host",
		"-e", gomock.Any(),
		"-e", gomock.Any(),
		"-e", gomock.Any(),
		"-e", gomock.Any(),
		"-e", gomock.Any(),
	)

	err := s.Install(ctx, getTinkBundle(), testIP, cluster.KubeconfigFile, stack.WithBootsOnDocker())
	if err != nil {
		t.Fatalf("failed to install Tinkerbell stack: %v", err)
	}
}

func TestTinkerbellStackUninstallLocalSucess(t *testing.T) {
	t.Setenv(features.TinkerbellProviderEnvVar, "true")
	mockCtrl := gomock.NewController(t)
	docker := mocks.NewMockDocker(mockCtrl)
	helm := mocks.NewMockHelm(mockCtrl)
	writer := filewritermocks.NewMockFileWriter(mockCtrl)
	ctx := context.Background()
	s := stack.NewInstaller(docker, writer, helm)

	docker.EXPECT().ForceRemove(ctx, boots)

	err := s.UninstallLocal(ctx)
	if err != nil {
		t.Fatalf("failed to install Tinkerbell stack: %v", err)
	}
}

func TestTinkerbellStackUninstallLocalFailure(t *testing.T) {
	t.Setenv(features.TinkerbellProviderEnvVar, "true")
	mockCtrl := gomock.NewController(t)
	docker := mocks.NewMockDocker(mockCtrl)
	helm := mocks.NewMockHelm(mockCtrl)
	writer := filewritermocks.NewMockFileWriter(mockCtrl)
	ctx := context.Background()
	s := stack.NewInstaller(docker, writer, helm)

	dockerError := "docker error"
	expectedError := fmt.Sprintf("removing local boots container: %s", dockerError)
	docker.EXPECT().ForceRemove(ctx, boots).Return(errors.New(dockerError))

	err := s.UninstallLocal(ctx)
	assert.EqualError(t, err, expectedError, "Error should be: %v, got: %v", expectedError, err)
}