/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package history

import (
	"context"
	"errors"
	"fmt"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/configMapAndSecret"
	read2 "github.com/devtron-labs/devtron/pkg/deployment/manifest/configMapAndSecret/read"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate"
	bean2 "github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/read"
	bean3 "github.com/devtron-labs/devtron/pkg/pipeline/history/bean"
	"github.com/devtron-labs/devtron/pkg/variables"
	repository5 "github.com/devtron-labs/devtron/pkg/variables/repository"
	"github.com/devtron-labs/devtron/util/sliceUtil"
	"go.opentelemetry.io/otel"
	"time"

	"github.com/devtron-labs/devtron/api/bean"
	repository2 "github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/pipeline/history/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

// TODO: Prakash, merge this interface with interface in deployment/manifest/deploymentTemplate/DeploymentTemplateHistoryService.go and extract out read logic in read repo
type DeployedConfigurationHistoryService interface {
	//TODO: rethink if the below method right at this place
	CreateHistoriesForDeploymentTrigger(ctx context.Context, pipeline *pipelineConfig.Pipeline, strategy *chartConfig.PipelineStrategy, envOverride *bean2.EnvConfigOverride, deployedOn time.Time, deployedBy int32) error

	GetDeployedConfigurationByWfrId(ctx context.Context, pipelineId, wfrId int) ([]*bean3.DeploymentConfigurationDto, error)
	GetDeployedHistoryComponentList(pipelineId, baseConfigId int, historyComponent, historyComponentName string) ([]*bean3.DeployedHistoryComponentMetadataDto, error)
	GetDeployedHistoryComponentDetail(ctx context.Context, pipelineId, id int, historyComponent, historyComponentName string, userHasAdminAccess bool) (*bean3.HistoryDetailDto, error)
	GetAllDeployedConfigurationByPipelineIdAndLatestWfrId(ctx context.Context, pipelineId int, userHasAdminAccess bool) (*bean3.AllDeploymentConfigurationDetail, error)
	GetAllDeployedConfigurationByPipelineIdAndWfrId(ctx context.Context, pipelineId, wfrId int, userHasAdminAccess bool) (*bean3.AllDeploymentConfigurationDetail, error)
	GetLatestDeployedArtifactByPipelineId(pipelineId int) (*repository2.CiArtifact, error)
}

type DeployedConfigurationHistoryServiceImpl struct {
	logger                               *zap.SugaredLogger
	userService                          user.UserService
	deploymentTemplateHistoryService     deploymentTemplate.DeploymentTemplateHistoryService
	strategyHistoryService               PipelineStrategyHistoryService
	configMapHistoryService              configMapAndSecret.ConfigMapHistoryService
	cdWorkflowRepository                 pipelineConfig.CdWorkflowRepository
	scopedVariableManager                variables.ScopedVariableCMCSManager
	deploymentTemplateHistoryReadService read.DeploymentTemplateHistoryReadService
	configMapHistoryReadService          read2.ConfigMapHistoryReadService
}

func NewDeployedConfigurationHistoryServiceImpl(logger *zap.SugaredLogger,
	userService user.UserService, deploymentTemplateHistoryService deploymentTemplate.DeploymentTemplateHistoryService,
	strategyHistoryService PipelineStrategyHistoryService, configMapHistoryService configMapAndSecret.ConfigMapHistoryService,
	cdWorkflowRepository pipelineConfig.CdWorkflowRepository,
	scopedVariableManager variables.ScopedVariableCMCSManager,
	deploymentTemplateHistoryReadService read.DeploymentTemplateHistoryReadService,
	configMapHistoryReadService read2.ConfigMapHistoryReadService,
) *DeployedConfigurationHistoryServiceImpl {
	return &DeployedConfigurationHistoryServiceImpl{
		logger:                               logger,
		userService:                          userService,
		deploymentTemplateHistoryService:     deploymentTemplateHistoryService,
		strategyHistoryService:               strategyHistoryService,
		configMapHistoryService:              configMapHistoryService,
		cdWorkflowRepository:                 cdWorkflowRepository,
		scopedVariableManager:                scopedVariableManager,
		deploymentTemplateHistoryReadService: deploymentTemplateHistoryReadService,
		configMapHistoryReadService:          configMapHistoryReadService,
	}
}

func (impl *DeployedConfigurationHistoryServiceImpl) CreateHistoriesForDeploymentTrigger(ctx context.Context, pipeline *pipelineConfig.Pipeline, strategy *chartConfig.PipelineStrategy, envOverride *bean2.EnvConfigOverride, deployedOn time.Time, deployedBy int32) error {
	_, span := otel.Tracer("orchestrator").Start(ctx, "DeployedConfigurationHistoryServiceImpl.CreateHistoriesForDeploymentTrigger")
	defer span.End()
	deploymentTemplateHistoryId, templateHistoryExists, err := impl.deploymentTemplateHistoryReadService.CheckIfTriggerHistoryExistsForPipelineIdOnTime(pipeline.Id, deployedOn)
	if err != nil {
		impl.logger.Errorw("error in checking if deployment template history exists for deployment trigger", "err", err)
		return err
	}
	if !templateHistoryExists {
		// creating history for deployment template
		deploymentTemplateHistory, err := impl.deploymentTemplateHistoryService.CreateDeploymentTemplateHistoryForDeploymentTrigger(pipeline, envOverride, envOverride.Chart.ImageDescriptorTemplate, deployedOn, deployedBy)
		if err != nil {
			impl.logger.Errorw("error in creating deployment template history for deployment trigger", "err", err)
			return err
		}
		deploymentTemplateHistoryId = deploymentTemplateHistory.Id
	}
	cmId, csId, cmCsHistoryExists, err := impl.configMapHistoryReadService.CheckIfTriggerHistoryExistsForPipelineIdOnTime(pipeline.Id, deployedOn)
	if err != nil {
		impl.logger.Errorw("error in checking if config map/ secrete history exists for deployment trigger", "err", err)
		return err
	}
	if !cmCsHistoryExists {
		cmId, csId, err = impl.configMapHistoryService.CreateCMCSHistoryForDeploymentTrigger(pipeline, deployedOn, deployedBy)
		if err != nil {
			impl.logger.Errorw("error in creating CM/CS history for deployment trigger", "err", err)
			return err
		}
	}
	if strategy != nil {
		// checking if pipeline strategy configuration for this pipelineId and with deployedOn time exists or not
		strategyHistoryExists, err := impl.strategyHistoryService.CheckIfTriggerHistoryExistsForPipelineIdOnTime(pipeline.Id, deployedOn)
		if err != nil {
			impl.logger.Errorw("error in checking if deployment template history exists for deployment trigger", "err", err)
			return err
		}
		if !strategyHistoryExists {
			err = impl.strategyHistoryService.CreateStrategyHistoryForDeploymentTrigger(strategy, deployedOn, deployedBy, pipeline.TriggerType)
			if err != nil {
				impl.logger.Errorw("error in creating strategy history for deployment trigger", "err", err)
				return err
			}
		}
	}

	var variableSnapshotHistories = sliceUtil.GetBeansPtr(
		repository5.GetSnapshotBean(deploymentTemplateHistoryId, repository5.HistoryReferenceTypeDeploymentTemplate, envOverride.VariableSnapshot),
		repository5.GetSnapshotBean(cmId, repository5.HistoryReferenceTypeConfigMap, envOverride.VariableSnapshotForCM),
		repository5.GetSnapshotBean(csId, repository5.HistoryReferenceTypeSecret, envOverride.VariableSnapshotForCS),
	)
	if len(variableSnapshotHistories) > 0 {
		err = impl.scopedVariableManager.SaveVariableHistoriesForTrigger(variableSnapshotHistories, deployedBy)
		if err != nil {
			return err
		}
	}
	return nil
}

func (impl *DeployedConfigurationHistoryServiceImpl) GetLatestDeployedArtifactByPipelineId(pipelineId int) (*repository2.CiArtifact, error) {
	wfr, err := impl.cdWorkflowRepository.FindLatestByPipelineIdAndRunnerType(pipelineId, bean.CD_WORKFLOW_TYPE_DEPLOY)
	if err != nil {
		impl.logger.Infow("error in getting latest deploy stage wfr by pipelineId", "err", err, "pipelineId", pipelineId)
		return nil, err
	}
	return wfr.CdWorkflow.CiArtifact, nil
}

func (impl *DeployedConfigurationHistoryServiceImpl) GetDeployedConfigurationByWfrId(ctx context.Context, pipelineId, wfrId int) ([]*bean3.DeploymentConfigurationDto, error) {
	newCtx, span := otel.Tracer("orchestrator").Start(ctx, "DeployedConfigurationHistoryServiceImpl.GetDeployedConfigurationByWfrId")
	defer span.End()
	var deployedConfigurations []*bean3.DeploymentConfigurationDto
	//checking if deployment template configuration for this pipelineId and wfrId exists or not
	templateHistoryId, exists, err := impl.deploymentTemplateHistoryReadService.CheckIfHistoryExistsForPipelineIdAndWfrId(pipelineId, wfrId)
	if err != nil {
		impl.logger.Errorw("error in checking if history exists for deployment template", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}

	deploymentTemplateConfiguration := &bean3.DeploymentConfigurationDto{
		Name: bean3.DEPLOYMENT_TEMPLATE_TYPE_HISTORY_COMPONENT,
	}
	if exists {
		deploymentTemplateConfiguration.Id = templateHistoryId
	}
	deployedConfigurations = append(deployedConfigurations, deploymentTemplateConfiguration)

	//checking if pipeline strategy configuration for this pipelineId and wfrId exists or not

	strategyHistoryId, exists, err := impl.strategyHistoryService.CheckIfHistoryExistsForPipelineIdAndWfrId(newCtx, pipelineId, wfrId)
	if err != nil {
		impl.logger.Errorw("error in checking if history exists for pipeline strategy", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	pipelineStrategyConfiguration := &bean3.DeploymentConfigurationDto{
		Name: bean3.PIPELINE_STRATEGY_TYPE_HISTORY_COMPONENT,
	}
	if exists {
		pipelineStrategyConfiguration.Id = strategyHistoryId
		deployedConfigurations = append(deployedConfigurations, pipelineStrategyConfiguration)
	}

	//checking if configmap history data exists and get its details
	configmapHistory, exists, names, err := impl.configMapHistoryReadService.GetDeployedHistoryByPipelineIdAndWfrId(pipelineId, wfrId, repository.CONFIGMAP_TYPE)
	if err != nil {
		impl.logger.Errorw("error in checking if history exists for configmap", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	if exists {
		configmapConfiguration := &bean3.DeploymentConfigurationDto{
			Id:                  configmapHistory.Id,
			Name:                bean3.CONFIGMAP_TYPE_HISTORY_COMPONENT,
			ChildComponentNames: names,
		}
		deployedConfigurations = append(deployedConfigurations, configmapConfiguration)
	}

	//checking if secret history data exists and get its details
	secretHistory, exists, names, err := impl.configMapHistoryReadService.GetDeployedHistoryByPipelineIdAndWfrId(pipelineId, wfrId, repository.SECRET_TYPE)
	if err != nil {
		impl.logger.Errorw("error in checking if history exists for secret", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	if exists {
		secretConfiguration := &bean3.DeploymentConfigurationDto{
			Id:                  secretHistory.Id,
			Name:                bean3.SECRET_TYPE_HISTORY_COMPONENT,
			ChildComponentNames: names,
		}
		deployedConfigurations = append(deployedConfigurations, secretConfiguration)
	}
	return deployedConfigurations, nil
}

func (impl *DeployedConfigurationHistoryServiceImpl) GetDeployedHistoryComponentList(pipelineId, baseConfigId int, historyComponent, historyComponentName string) ([]*bean3.DeployedHistoryComponentMetadataDto, error) {
	var historyList []*bean3.DeployedHistoryComponentMetadataDto
	var err error
	if historyComponent == string(bean3.DEPLOYMENT_TEMPLATE_TYPE_HISTORY_COMPONENT) {
		historyList, err = impl.deploymentTemplateHistoryReadService.GetDeployedHistoryList(pipelineId, baseConfigId)
	} else if historyComponent == string(bean3.PIPELINE_STRATEGY_TYPE_HISTORY_COMPONENT) {
		historyList, err = impl.strategyHistoryService.GetDeployedHistoryList(pipelineId, baseConfigId)
	} else if historyComponent == string(bean3.CONFIGMAP_TYPE_HISTORY_COMPONENT) {
		historyList, err = impl.configMapHistoryReadService.GetDeployedHistoryList(pipelineId, baseConfigId, repository.CONFIGMAP_TYPE, historyComponentName)
	} else if historyComponent == string(bean3.SECRET_TYPE_HISTORY_COMPONENT) {
		historyList, err = impl.configMapHistoryReadService.GetDeployedHistoryList(pipelineId, baseConfigId, repository.SECRET_TYPE, historyComponentName)
	} else {
		return nil, errors.New(fmt.Sprintf("history of %s not supported", historyComponent))
	}
	if err != nil {
		impl.logger.Errorw("error in getting deployed history component list", "err", err, "pipelineId", pipelineId, "historyComponent", historyComponent, "componentName", historyComponentName)
		return nil, err
	}
	return historyList, nil
}

func (impl *DeployedConfigurationHistoryServiceImpl) GetDeployedHistoryComponentDetail(ctx context.Context, pipelineId, id int, historyComponent, historyComponentName string, userHasAdminAccess bool) (*bean3.HistoryDetailDto, error) {
	history := &bean3.HistoryDetailDto{}
	var err error
	if historyComponent == string(bean3.DEPLOYMENT_TEMPLATE_TYPE_HISTORY_COMPONENT) {
		history, err = impl.deploymentTemplateHistoryReadService.GetHistoryForDeployedTemplateById(ctx, id, pipelineId)
	} else if historyComponent == string(bean3.PIPELINE_STRATEGY_TYPE_HISTORY_COMPONENT) {
		history, err = impl.strategyHistoryService.GetHistoryForDeployedStrategyById(id, pipelineId)
	} else if historyComponent == string(bean3.CONFIGMAP_TYPE_HISTORY_COMPONENT) {
		history, err = impl.configMapHistoryReadService.GetHistoryForDeployedCMCSById(ctx, id, pipelineId, repository.CONFIGMAP_TYPE, historyComponentName, userHasAdminAccess)
	} else if historyComponent == string(bean3.SECRET_TYPE_HISTORY_COMPONENT) {
		history, err = impl.configMapHistoryReadService.GetHistoryForDeployedCMCSById(ctx, id, pipelineId, repository.SECRET_TYPE, historyComponentName, userHasAdminAccess)
	} else {
		return nil, errors.New(fmt.Sprintf("history of %s not supported", historyComponent))
	}
	if err != nil {
		impl.logger.Errorw("error in getting deployed history component detail", "err", err, "pipelineId", pipelineId, "id", id, "historyComponent", historyComponent, "componentName", historyComponentName)
		return nil, err
	}
	return history, nil
}

func (impl *DeployedConfigurationHistoryServiceImpl) GetAllDeployedConfigurationByPipelineIdAndLatestWfrId(ctx context.Context, pipelineId int, userHasAdminAccess bool) (*bean3.AllDeploymentConfigurationDetail, error) {
	//getting latest wfr from pipelineId
	wfr, err := impl.cdWorkflowRepository.FindLatestByPipelineIdAndRunnerType(pipelineId, bean.CD_WORKFLOW_TYPE_DEPLOY)
	if err != nil {
		impl.logger.Errorw("error in getting latest deploy stage wfr by pipelineId", "err", err, "pipelineId", pipelineId)
		return nil, err
	}
	deployedConfig, err := impl.GetAllDeployedConfigurationByPipelineIdAndWfrId(ctx, pipelineId, wfr.Id, userHasAdminAccess)
	if err != nil {
		impl.logger.Errorw("error in getting GetAllDeployedConfigurationByPipelineIdAndWfrId", "err", err, "pipelineID", pipelineId, "wfrId", wfr.Id)
		return nil, err
	}
	deployedConfig.WfrId = wfr.Id
	return deployedConfig, nil
}
func (impl *DeployedConfigurationHistoryServiceImpl) GetAllDeployedConfigurationByPipelineIdAndWfrId(ctx context.Context, pipelineId, wfrId int, userHasAdminAccess bool) (*bean3.AllDeploymentConfigurationDetail, error) {
	//getting history of deployment template for latest deployment
	deploymentTemplateHistory, err := impl.deploymentTemplateHistoryReadService.GetDeployedHistoryByPipelineIdAndWfrId(ctx, pipelineId, wfrId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting deployment template history by pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	//getting history of config map for latest deployment
	configMapHistory, err := impl.configMapHistoryReadService.GetDeployedHistoryDetailForCMCSByPipelineIdAndWfrId(ctx, pipelineId, wfrId, repository.CONFIGMAP_TYPE, userHasAdminAccess)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting config map history by pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	//getting history of secret for latest deployment
	secretHistory, err := impl.configMapHistoryReadService.GetDeployedHistoryDetailForCMCSByPipelineIdAndWfrId(ctx, pipelineId, wfrId, repository.SECRET_TYPE, userHasAdminAccess)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting secret history by pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	//getting history of pipeline strategy for latest deployment
	strategyHistory, err := impl.strategyHistoryService.GetLatestDeployedHistoryByPipelineIdAndWfrId(ctx, pipelineId, wfrId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting strategy history by pipelineId and wfrId", "err", err, "pipelineId", pipelineId, "wfrId", wfrId)
		return nil, err
	}
	allDeploymentConfigurationHistoryDetail := &bean3.AllDeploymentConfigurationDetail{
		DeploymentTemplateConfig: deploymentTemplateHistory,
		ConfigMapConfig:          configMapHistory,
		SecretConfig:             secretHistory,
		StrategyConfig:           strategyHistory,
	}
	return allDeploymentConfigurationHistoryDetail, nil
}
