/*
 * Copyright (c) 2020-2024. Devtron Inc.
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

package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/devtron/internal/sql/models"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/internal/util"
	chartRepoRepository "github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	repository3 "github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"github.com/devtron-labs/devtron/pkg/commonService"
	history2 "github.com/devtron-labs/devtron/pkg/deployment/manifest/configMapAndSecret"
	"github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/history/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/variables"
	util2 "github.com/devtron-labs/devtron/util"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
	KubernetesSecret  string = "KubernetesSecret"
	AWSSecretsManager string = "AWSSecretsManager"
	AWSSystemManager  string = "AWSSystemManager"
	HashiCorpVault    string = "HashiCorpVault"
)

type ConfigMapService interface {
	CMGlobalAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error)
	CMGlobalFetch(appId int) (*bean.ConfigDataRequest, error)
	CMEnvironmentAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error)
	CMEnvironmentFetch(appId int, envId int) (*bean.ConfigDataRequest, error)
	CMGlobalFetchForEdit(name string, id int) (*bean.ConfigDataRequest, error)
	CMEnvironmentFetchForEdit(name string, id int, appId int, envId int) (*bean.ConfigDataRequest, error)

	CmCsConfigGlobalFetchUsingAppId(name string, appId int, resourceType bean.ResourceType) (*bean.ConfigDataRequest, error)
	CmCsConfigOverrideFetchUsingAppAndEnvId(name string, appId, envId int, resourceType bean.ResourceType) (*bean.ConfigDataRequest, error)

	CSGlobalAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error)
	CSGlobalFetch(appId int) (*bean.ConfigDataRequest, error)
	CSEnvironmentAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error)
	CSEnvironmentFetch(appId int, envId int) (*bean.ConfigDataRequest, error)

	CMGlobalDelete(name string, id int, userId int32) (bool, error)
	CMEnvironmentDelete(name string, id int, userId int32) (bool, error)
	CSGlobalDelete(name string, id int, userId int32) (bool, error)
	CSEnvironmentDelete(name string, id int, userId int32) (bool, error)

	CMGlobalDeleteByAppId(name string, appId int, userId int32) (bool, error)
	CMEnvironmentDeleteByAppIdAndEnvId(name string, appId int, envId int, userId int32) (bool, error)
	CSGlobalDeleteByAppId(name string, appId int, userId int32) (bool, error)
	CSEnvironmentDeleteByAppIdAndEnvId(name string, appId int, envId int, userId int32) (bool, error)

	CSGlobalFetchForEdit(name string, id int) (*bean.ConfigDataRequest, error)
	CSEnvironmentFetchForEdit(name string, id int, appId int, envId int) (*bean.ConfigDataRequest, error)
	ConfigSecretGlobalBulkPatch(bulkPatchRequest *bean.BulkPatchRequest) (*bean.BulkPatchRequest, error)
	ConfigSecretEnvironmentBulkPatch(bulkPatchRequest *bean.BulkPatchRequest) (*bean.BulkPatchRequest, error)

	ConfigSecretEnvironmentCreate(createJobEnvOverrideRequest *bean.CreateJobEnvOverridePayload) (*bean.CreateJobEnvOverridePayload, error)
	ConfigSecretEnvironmentDelete(createJobEnvOverrideRequest *bean.CreateJobEnvOverridePayload) (*bean.CreateJobEnvOverridePayload, error)
	ConfigSecretEnvironmentGet(appId int) ([]bean.JobEnvOverrideResponse, error)
	ConfigSecretEnvironmentClone(appId int, cloneAppId int, userId int32) ([]chartConfig.ConfigMapEnvModel, error)

	FetchCmCsNamesAppAndEnvLevel(appId int, envId int) ([]bean.ConfigNameAndType, []bean.ConfigNameAndType, error)
}

type ConfigMapServiceImpl struct {
	chartRepository          chartRepoRepository.ChartRepository
	logger                   *zap.SugaredLogger
	repoRepository           chartRepoRepository.ChartRepoRepository
	mergeUtil                util.MergeUtil
	pipelineConfigRepository chartConfig.PipelineConfigRepository
	configMapRepository      chartConfig.ConfigMapRepository
	commonService            commonService.CommonService
	appRepository            app.AppRepository
	configMapHistoryService  history2.ConfigMapHistoryService
	environmentRepository    repository3.EnvironmentRepository
	scopedVariableManager    variables.ScopedVariableCMCSManager
}

func NewConfigMapServiceImpl(chartRepository chartRepoRepository.ChartRepository,
	logger *zap.SugaredLogger,
	repoRepository chartRepoRepository.ChartRepoRepository,
	mergeUtil util.MergeUtil,
	pipelineConfigRepository chartConfig.PipelineConfigRepository,
	configMapRepository chartConfig.ConfigMapRepository,
	commonService commonService.CommonService, appRepository app.AppRepository,
	configMapHistoryService history2.ConfigMapHistoryService, environmentRepository repository3.EnvironmentRepository,
	scopedVariableManager variables.ScopedVariableCMCSManager,
) *ConfigMapServiceImpl {
	return &ConfigMapServiceImpl{
		chartRepository:          chartRepository,
		logger:                   logger,
		repoRepository:           repoRepository,
		mergeUtil:                mergeUtil,
		pipelineConfigRepository: pipelineConfigRepository,
		configMapRepository:      configMapRepository,
		commonService:            commonService,
		appRepository:            appRepository,
		configMapHistoryService:  configMapHistoryService,
		environmentRepository:    environmentRepository,
		scopedVariableManager:    scopedVariableManager,
	}
}

func (impl ConfigMapServiceImpl) checkIfConfigDataAlreadyExist(appId int) (int, error) {
	config, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error while checking if config data exist from db by appId", "appId", appId, "error", err)
		return 0, err
	} else if util.IsErrNoRows(err) {
		return 0, nil
	}
	return config.Id, nil
}

func (impl ConfigMapServiceImpl) CMGlobalAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error) {
	if len(configMapRequest.ConfigData) != 1 {
		return nil, fmt.Errorf("invalid request multiple config found for add or update")
	}
	configData := configMapRequest.ConfigData[0]
	valid, err := impl.validateConfigData(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}
	var model *chartConfig.ConfigMapAppModel
	requestId, err := impl.checkIfConfigDataAlreadyExist(configMapRequest.AppId)
	if err != nil {
		impl.logger.Errorw("error in checking if config map data already exists or not for appId", "appId", configMapRequest.AppId, "error", err)
		return configMapRequest, err
	}
	if requestId > 0 {
		configMapRequest.Id = requestId
	}
	if configMapRequest.Id > 0 {
		model, err = impl.configMapRepository.GetByIdAppLevel(configMapRequest.Id)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		configsList := &bean.ConfigsList{}
		found := false
		var configs []*bean.ConfigData
		if len(model.ConfigMapData) > 0 {
			err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
			if err != nil {
				impl.logger.Debugw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range configsList.ConfigData {
			if item.Name == configData.Name {
				item.Data = configData.Data
				item.MountPath = configData.MountPath
				item.Type = configData.Type
				item.External = configData.External
				item.ExternalSecretType = configData.ExternalSecretType
				found = true
				item.SubPath = configData.SubPath
				item.FilePermission = configData.FilePermission
			}
			configs = append(configs, item)
		}

		if !found {
			configs = append(configs, configData)
		}
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return nil, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = configMapRequest.UserId
		model.UpdatedOn = time.Now()
		configMap, err := impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id

	} else {
		//creating config map record for first time
		configsList := &bean.ConfigsList{
			ConfigData: configMapRequest.ConfigData,
		}
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return nil, err
		}
		model = &chartConfig.ConfigMapAppModel{
			AppId:         configMapRequest.AppId,
			ConfigMapData: string(configDataByte),
		}
		model.CreatedBy = configMapRequest.UserId
		model.UpdatedBy = configMapRequest.UserId
		model.CreatedOn = time.Now()
		model.UpdatedOn = time.Now()

		configMap, err := impl.configMapRepository.CreateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating app level", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id
	}
	//VARIABLE_MAPPING_UPDATE
	err = impl.scopedVariableManager.CreateVariableMappingsForCMApp(model)
	if err != nil {
		return nil, err
	}
	err = impl.configMapHistoryService.CreateHistoryFromAppLevelConfig(model, repository.CONFIGMAP_TYPE)
	if err != nil {
		impl.logger.Errorw("error in creating entry for configmap history", "err", err)
		return nil, err
	}
	return configMapRequest, nil
}

func (impl ConfigMapServiceImpl) CMGlobalFetch(appId int) (*bean.ConfigDataRequest, error) {
	configMapGlobal, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Debugw("no config map data found for this request", "appId", appId)
	}

	configMapGlobalList := &bean.ConfigsList{}
	if len(configMapGlobal.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(configMapGlobal.ConfigMapData), configMapGlobalList)
		if err != nil {
			impl.logger.Debugw("error while Unmarshal", "error", err)
		}
	}
	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapGlobal.Id
	configDataRequest.AppId = appId
	//configDataRequest.ConfigData = configMapGlobalList.ConfigData
	for _, item := range configMapGlobalList.ConfigData {
		item.Global = true
		configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
	}
	if configDataRequest.ConfigData == nil {
		list := []*bean.ConfigData{}
		configDataRequest.ConfigData = list
	} else {
		//configDataRequest.ConfigData = configMapGlobalList.ConfigData
	}

	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CMEnvironmentAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error) {

	if len(configMapRequest.ConfigData) != 1 {
		return nil, fmt.Errorf("invalid request multiple config found for add or update")
	}
	configData := configMapRequest.ConfigData[0]
	valid, err := impl.validateConfigData(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}

	appLevelConfigMap, err := impl.CmCsConfigGlobalFetchUsingAppId(configData.Name, configMapRequest.AppId, bean.CM)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching app level config", "appId", configMapRequest.AppId, "err", err)
		return configMapRequest, err
	}

	var model *chartConfig.ConfigMapEnvModel
	if configMapRequest.Id > 0 {
		model, err = impl.configMapRepository.GetByIdEnvLevel(configMapRequest.Id)
	} else if configMapRequest.AppId > 0 && configMapRequest.EnvironmentId > 0 {
		model, err = impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(configMapRequest.AppId, configMapRequest.EnvironmentId)
	}
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if err == nil && model.Id > 0 {
		configsList := &bean.ConfigsList{}
		found := false
		var configs []*bean.ConfigData
		if len(model.ConfigMapData) > 0 {
			err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
			if err != nil {
				impl.logger.Debugw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range configsList.ConfigData {
			if item.Name == configData.Name {
				item.Data = configData.Data
				item.MountPath = configData.MountPath
				item.Type = configData.Type
				item.External = configData.External
				item.ExternalSecretType = configData.ExternalSecretType
				item.SubPath = configData.SubPath
				item.FilePermission = configData.FilePermission
				found = true
				item.MergeStrategy = configData.MergeStrategy
				if len(appLevelConfigMap.ConfigData) > 0 {
					if len(item.MergeStrategy) == 0 {
						item.MergeStrategy = models.MERGE_STRATEGY_REPLACE
					}
				}
			}
			configs = append(configs, item)
		}

		if !found {
			configs = append(configs, configData)
		}
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return nil, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = configMapRequest.UserId
		model.UpdatedOn = time.Now()

		configMap, err := impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id

	} else if err == pg.ErrNoRows {
		//creating config map record for first time
		configsList := &bean.ConfigsList{
			ConfigData: configMapRequest.ConfigData,
		}
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return nil, err
		}
		model = &chartConfig.ConfigMapEnvModel{
			AppId:         configMapRequest.AppId,
			EnvironmentId: configMapRequest.EnvironmentId,
			ConfigMapData: string(configDataByte),
		}
		model.CreatedBy = configMapRequest.UserId
		model.UpdatedBy = configMapRequest.UserId

		configMap, err := impl.configMapRepository.CreateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating app level", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id
	}
	//VARIABLE_MAPPING_UPDATE
	//err = impl.extractAndMapVariables(model.ConfigMapData, model.Id, repository5.EntityTypeConfigMapEnvLevel, configMapRequest.UserId)
	err = impl.scopedVariableManager.CreateVariableMappingsForCMEnv(model)
	if err != nil {
		return nil, err
	}
	err = impl.configMapHistoryService.CreateHistoryFromEnvLevelConfig(model, repository.CONFIGMAP_TYPE)
	if err != nil {
		impl.logger.Errorw("error in creating entry for CM/CS history in bulk update", "err", err)
		return nil, err
	}
	return configMapRequest, nil
}

func (impl ConfigMapServiceImpl) CMGlobalFetchForEdit(name string, id int) (*bean.ConfigDataRequest, error) {
	configMapGlobal, err := impl.configMapRepository.GetByIdAppLevel(id)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Debugw("no config map data found for this request", "id", id)
	}

	configMapGlobalList := &bean.ConfigsList{}
	if len(configMapGlobal.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(configMapGlobal.ConfigMapData), configMapGlobalList)
		if err != nil {
			impl.logger.Debugw("error while Unmarshal", "error", err)
		}
	}
	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapGlobal.Id
	for _, item := range configMapGlobalList.ConfigData {
		if item.Name == name {
			item.Global = true
			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
			break
		}
	}
	if configDataRequest.ConfigData == nil {
		list := []*bean.ConfigData{}
		configDataRequest.ConfigData = list
	}

	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CMEnvironmentFetchForEdit(name string, id int, appId int, envId int) (*bean.ConfigDataRequest, error) {
	configDataRequest, err := impl.CMEnvironmentFetch(appId, envId)
	if err != nil {
		return nil, err
	}
	var configs []*bean.ConfigData
	for _, configData := range configDataRequest.ConfigData {
		if configData.Name == name {
			configs = append(configs, configData)
		}
	}
	configDataRequest.ConfigData = configs
	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CMEnvironmentFetch(appId int, envId int) (*bean.ConfigDataRequest, error) {
	configMapGlobal, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Debugw("no config map data found for this request", "appId", appId)
	}
	configMapGlobalList := &bean.ConfigsList{}
	if len(configMapGlobal.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(configMapGlobal.ConfigMapData), configMapGlobalList)
		if err != nil {
			impl.logger.Errorw("error while Unmarshal", "error", err)
		}
	}
	configMapEnvLevel, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(appId, envId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Debugw("no config map data found for this request", "appId", appId)
	}
	configsListEnvLevel := &bean.ConfigsList{}
	if len(configMapEnvLevel.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(configMapEnvLevel.ConfigMapData), configsListEnvLevel)
		if err != nil {
			impl.logger.Debugw("error while Unmarshal", "error", err)
		}
		processCmCsEnvLevel(configsListEnvLevel.ConfigData)
	}
	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapEnvLevel.Id
	configDataRequest.AppId = appId
	configDataRequest.EnvironmentId = envId
	//configDataRequest.ConfigData = configsListEnvLevel.ConfigData

	kv1 := make(map[string]json.RawMessage)
	kv11 := make(map[string]string)
	kv2 := make(map[string]json.RawMessage)
	for _, item := range configMapGlobalList.ConfigData {
		kv1[item.Name] = item.Data
		kv11[item.Name] = item.MountPath
	}
	for _, item := range configsListEnvLevel.ConfigData {
		kv2[item.Name] = item.Data
	}

	//add those items which are in global only
	for _, item := range configMapGlobalList.ConfigData {
		if _, ok := kv2[item.Name]; !ok {
			item.Global = true
			item.DefaultData = item.Data
			item.DefaultMountPath = item.MountPath
			item.Data = nil
			item.MountPath = ""
			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		}
	}

	//add all the items from environment level, add default data to items which are override from global
	for _, item := range configsListEnvLevel.ConfigData {
		if val, ok := kv1[item.Name]; ok {
			item.DefaultData = val
			item.DefaultMountPath = kv11[item.Name]
			item.Global = true
			item.Overridden = true

			if len(item.MergeStrategy) == 0 {
				item.MergeStrategy = models.MERGE_STRATEGY_REPLACE
			}

			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		} else {
			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		}
	}

	if configDataRequest.ConfigData == nil {
		list := []*bean.ConfigData{}
		configDataRequest.ConfigData = list
	}

	return configDataRequest, nil
}
func processCmCsEnvLevel(configData []*bean.ConfigData) {
	for index, _ := range configData {
		configData[index].Global = false
	}
}

// ---------------------------------------------------------------------------------------------

func (impl ConfigMapServiceImpl) CSGlobalAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error) {
	if len(configMapRequest.ConfigData) != 1 {
		return nil, fmt.Errorf("invalid request multiple config found for add or update")
	}
	configData := configMapRequest.ConfigData[0]
	// validating config/secret data at service layer since this func is consumed in multiple flows, hence preventing code duplication
	valid, err := impl.validateConfigData(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}

	valid, err = impl.validateConfigDataForSecretsOnly(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating secrets only data", "error", err)
		return configMapRequest, err
	}

	valid, err = impl.validateExternalSecretChartCompatibility(configMapRequest.AppId, configMapRequest.EnvironmentId, configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}
	requestId, err := impl.checkIfConfigDataAlreadyExist(configMapRequest.AppId)
	if err != nil {
		impl.logger.Errorw("error in checking if config secret data already exists or not for appId", "appId", configMapRequest.AppId, "error", err)
		return configMapRequest, err
	}
	if requestId > 0 {
		configMapRequest.Id = requestId
	}
	var model *chartConfig.ConfigMapAppModel
	if configMapRequest.Id > 0 {
		model, err = impl.configMapRepository.GetByIdAppLevel(configMapRequest.Id)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		secretsList := &bean.SecretsList{}
		found := false
		var configs []*bean.ConfigData
		if len(model.SecretData) > 0 {
			err = json.Unmarshal([]byte(model.SecretData), secretsList)
			if err != nil {
				impl.logger.Debugw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range secretsList.ConfigData {
			if item.Name == configData.Name {
				found = true
				item.Data = configData.Data
				item.MountPath = configData.MountPath
				item.Type = configData.Type
				item.External = configData.External
				item.ExternalSecretType = configData.ExternalSecretType
				item.ESOSecretData = configData.ESOSecretData
				item.ExternalSecret = configData.ExternalSecret
				item.RoleARN = configData.RoleARN
				item.SubPath = configData.SubPath
				item.ESOSubPath = configData.ESOSubPath
				item.FilePermission = configData.FilePermission
			}
			configs = append(configs, item)
		}

		if !found {
			configs = append(configs, configData)
		}
		secretsList.ConfigData = configs
		configDataByte, err := json.Marshal(secretsList)
		if err != nil {
			return nil, err
		}
		model.SecretData = string(configDataByte)
		model.UpdatedBy = configMapRequest.UserId
		model.UpdatedOn = time.Now()

		secret, err := impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		configMapRequest.Id = secret.Id

	} else {
		//creating config map record for first time
		secretsList := &bean.SecretsList{
			ConfigData: configMapRequest.ConfigData,
		}
		secretDataByte, err := json.Marshal(secretsList)
		if err != nil {
			return nil, err
		}
		model = &chartConfig.ConfigMapAppModel{
			AppId:      configMapRequest.AppId,
			SecretData: string(secretDataByte),
		}
		model.CreatedBy = configMapRequest.UserId
		model.UpdatedBy = configMapRequest.UserId
		model.CreatedOn = time.Now()
		model.UpdatedOn = time.Now()

		secret, err := impl.configMapRepository.CreateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating app level", "error", err)
			return nil, err
		}
		configMapRequest.Id = secret.Id
	}
	//VARIABLE_MAPPING_UPDATE
	//sl := bean.SecretsList{}
	//data, err := sl.GetTransformedDataForSecretList(model.SecretData, util2.DecodeSecret)
	//if err != nil {
	//	return nil, err
	//}
	//err = impl.extractAndMapVariables(data, model.Id, repository5.EntityTypeSecretAppLevel, configMapRequest.UserId)
	err = impl.scopedVariableManager.CreateVariableMappingsForSecretApp(model)
	if err != nil {
		return nil, err
	}
	err = impl.configMapHistoryService.CreateHistoryFromAppLevelConfig(model, repository.SECRET_TYPE)
	if err != nil {
		impl.logger.Errorw("error in creating entry for secret history", "err", err)
		return nil, err
	}
	return configMapRequest, nil
}

func (impl ConfigMapServiceImpl) CSGlobalFetch(appId int) (*bean.ConfigDataRequest, error) {
	configMapGlobal, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Warnw("no app level secret found for this request", "appId", appId)
	}

	configMapGlobalList := &bean.SecretsList{}
	if len(configMapGlobal.SecretData) > 0 {
		err = json.Unmarshal([]byte(configMapGlobal.SecretData), configMapGlobalList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapGlobal.Id
	configDataRequest.AppId = appId

	for _, item := range configMapGlobalList.ConfigData {
		item.Global = true
		configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
	}

	if configDataRequest.ConfigData == nil {
		list := []*bean.ConfigData{}
		configDataRequest.ConfigData = list
	}

	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CSEnvironmentAddUpdate(configMapRequest *bean.ConfigDataRequest) (*bean.ConfigDataRequest, error) {
	if len(configMapRequest.ConfigData) != 1 {
		return nil, fmt.Errorf("invalid request multiple config found for add or update")
	}

	configData := configMapRequest.ConfigData[0]
	// validating config/secret data at service layer since this func is consumed in multiple flows, hence preventing code duplication
	valid, err := impl.validateConfigData(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}
	valid, err = impl.validateConfigDataForSecretsOnly(configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating secrets only data", "error", err)
		return configMapRequest, err
	}

	valid, err = impl.validateExternalSecretChartCompatibility(configMapRequest.AppId, configMapRequest.EnvironmentId, configData)
	if err != nil && !valid {
		impl.logger.Errorw("error in validating", "error", err)
		return configMapRequest, err
	}

	appLevelSecret, err := impl.CmCsConfigGlobalFetchUsingAppId(configData.Name, configMapRequest.AppId, bean.CS)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching app level config", "appId", configMapRequest.AppId, "err", err)
		return configMapRequest, err
	}

	var model *chartConfig.ConfigMapEnvModel
	if configMapRequest.Id > 0 {
		model, err = impl.configMapRepository.GetByIdEnvLevel(configMapRequest.Id)
	} else if configMapRequest.AppId > 0 && configMapRequest.EnvironmentId > 0 {
		model, err = impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(configMapRequest.AppId, configMapRequest.EnvironmentId)
	}
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if err == nil && model.Id > 0 {
		configsList := &bean.SecretsList{}
		found := false
		var configs []*bean.ConfigData
		if len(model.SecretData) > 0 {
			err = json.Unmarshal([]byte(model.SecretData), configsList)
			if err != nil {
				impl.logger.Warnw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range configsList.ConfigData {
			if item.Name == configData.Name {
				item.Data = configData.Data
				item.MountPath = configData.MountPath
				item.Type = configData.Type
				item.External = configData.External
				item.ExternalSecretType = configData.ExternalSecretType
				item.ESOSecretData = configData.ESOSecretData
				item.ExternalSecret = configData.ExternalSecret
				item.RoleARN = configData.RoleARN
				item.SubPath = configData.SubPath
				item.ESOSubPath = configData.ESOSubPath
				item.FilePermission = configData.FilePermission
				found = true
				item.MergeStrategy = configData.MergeStrategy
				if len(appLevelSecret.ConfigData) > 0 {
					if len(item.MergeStrategy) == 0 {
						item.MergeStrategy = models.MERGE_STRATEGY_REPLACE
					}
				}
			}
			configs = append(configs, item)
		}

		if !found {
			configs = append(configs, configData)
		}
		configsList.ConfigData = configs
		secretDataByte, err := json.Marshal(configsList)
		if err != nil {
			return nil, err
		}
		model.SecretData = string(secretDataByte)
		model.UpdatedBy = configMapRequest.UserId
		model.UpdatedOn = time.Now()

		configMap, err := impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id

	} else if err == pg.ErrNoRows {
		//creating config map record for first time
		secretsList := &bean.SecretsList{
			ConfigData: configMapRequest.ConfigData,
		}
		secretDataByte, err := json.Marshal(secretsList)
		if err != nil {
			return nil, err
		}
		model = &chartConfig.ConfigMapEnvModel{
			AppId:         configMapRequest.AppId,
			EnvironmentId: configMapRequest.EnvironmentId,
			SecretData:    string(secretDataByte),
		}
		model.CreatedBy = configMapRequest.UserId
		model.UpdatedBy = configMapRequest.UserId

		configMap, err := impl.configMapRepository.CreateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating app level", "error", err)
			return nil, err
		}
		configMapRequest.Id = configMap.Id
	}
	err = impl.scopedVariableManager.CreateVariableMappingsForSecretEnv(model)
	if err != nil {
		return nil, err
	}
	err = impl.configMapHistoryService.CreateHistoryFromEnvLevelConfig(model, repository.SECRET_TYPE)
	if err != nil {
		impl.logger.Errorw("error in creating entry for CM/CS history in bulk update", "err", err)
		return nil, err
	}
	return configMapRequest, nil
}

func (impl ConfigMapServiceImpl) CSEnvironmentFetch(appId int, envId int) (*bean.ConfigDataRequest, error) {
	configMapGlobal, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Warnw("no app level secret found for this request", "appId", appId)
	}
	configMapGlobalList := &bean.SecretsList{}
	if len(configMapGlobal.SecretData) > 0 {
		err = json.Unmarshal([]byte(configMapGlobal.SecretData), configMapGlobalList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}

	configMapEnvLevel, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(appId, envId)
	if err != nil && pg.ErrNoRows != err {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if pg.ErrNoRows == err {
		impl.logger.Warnw("no env level secret found for this request", "appId", appId)
	}
	configsListEnvLevel := &bean.SecretsList{}
	if len(configMapEnvLevel.SecretData) > 0 {
		err = json.Unmarshal([]byte(configMapEnvLevel.SecretData), configsListEnvLevel)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
		processCmCsEnvLevel(configsListEnvLevel.ConfigData)
	}
	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapEnvLevel.Id
	configDataRequest.AppId = appId
	configDataRequest.EnvironmentId = envId

	//configDataRequest.ConfigData = configsListEnvLevel.ConfigData
	//var configs []ConfigData
	kv1 := make(map[string]json.RawMessage)
	kv11 := make(map[string]string)
	kv2 := make(map[string]json.RawMessage)

	kv1External := make(map[string][]bean.ExternalSecret)
	kv2External := make(map[string][]bean.ExternalSecret)

	kv1ESOSecret := make(map[string]bean.ESOSecretData)
	kv2ESOSecret := make(map[string]bean.ESOSecretData)
	for _, item := range configMapGlobalList.ConfigData {
		kv1[item.Name] = item.Data
		kv11[item.Name] = item.MountPath
		kv1External[item.Name] = item.ExternalSecret
		kv1ESOSecret[item.Name] = item.ESOSecretData
	}
	for _, item := range configsListEnvLevel.ConfigData {
		kv2[item.Name] = item.Data
		kv2External[item.Name] = item.ExternalSecret
		kv2ESOSecret[item.Name] = item.ESOSecretData
	}

	//add those items which are in global only
	for _, item := range configMapGlobalList.ConfigData {
		if _, ok := kv2[item.Name]; !ok {
			item.Global = true
			if item.Data != nil && item.ExternalSecret == nil {
				item.DefaultData = item.Data
			} else if item.ExternalSecret != nil {
				/*bytes, err := json.Marshal(item.ExternalSecret)
				if err != nil {

				}
				item.DefaultData = bytes*/
				item.DefaultExternalSecret = item.ExternalSecret
			}
			item.DefaultESOSecretData = item.ESOSecretData
			item.ESOSecretData.ESOData = nil
			item.ESOSecretData.SecretStore = nil
			item.ESOSecretData.ESODataFrom = nil
			item.ESOSecretData.Template = nil
			item.ESOSecretData.SecretStoreRef = nil
			item.ESOSecretData.RefreshInterval = ""
			item.DefaultMountPath = item.MountPath
			item.Data = nil
			item.ExternalSecret = nil
			item.MountPath = ""
			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		}
	}

	//add all the items from environment level, add default data to items which are override from global
	for _, item := range configsListEnvLevel.ConfigData {
		if val, ok := kv1[item.Name]; ok {
			item.DefaultData = val
			item.DefaultExternalSecret = kv1External[item.Name]
			item.DefaultMountPath = kv11[item.Name]
			item.Global = true
			item.Overridden = true
			item.DefaultESOSecretData = kv1ESOSecret[item.Name]

			if len(item.MergeStrategy) == 0 {
				item.MergeStrategy = models.MERGE_STRATEGY_REPLACE
			}

			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		} else {
			configDataRequest.ConfigData = append(configDataRequest.ConfigData, item)
		}
	}

	if configDataRequest.ConfigData == nil {
		list := []*bean.ConfigData{}
		configDataRequest.ConfigData = list
	} else {
		//configDataRequest.ConfigData = configMapGlobalList.ConfigData
	}
	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CMGlobalDelete(name string, id int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByIdAppLevel(id)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.ConfigsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()

		_, err = impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at app level", "error", err)
			return false, err
		}
		err = impl.scopedVariableManager.CreateVariableMappingsForCMApp(model)
		if err != nil {
			return false, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromAppLevelConfig(model, repository.CONFIGMAP_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for configmap history", "err", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)

	}

	return true, nil
}

func (impl ConfigMapServiceImpl) CMEnvironmentDelete(name string, id int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByIdEnvLevel(id)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.ConfigsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		//VARIABLE_MAPPING_UPDATE
		err = impl.scopedVariableManager.CreateVariableMappingsForCMEnv(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at env level", "error", err)
			return false, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromEnvLevelConfig(model, repository.CONFIGMAP_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for configmap env history", "err", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)
	}
	return true, nil
}

func (impl ConfigMapServiceImpl) CSGlobalDelete(name string, id int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByIdAppLevel(id)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.SecretsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.SecretData) > 0 {
		err = json.Unmarshal([]byte(model.SecretData), configsList)
		if err != nil {
			impl.logger.Debugw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.SecretData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		//VARIABLE_MAPPING_UPDATE
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretApp(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at app level", "error", err)
			return false, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromAppLevelConfig(model, repository.SECRET_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for secret history", "err", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)

	}

	return true, nil
}

func (impl ConfigMapServiceImpl) CSEnvironmentDelete(name string, id int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByIdEnvLevel(id)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.SecretsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.SecretData) > 0 {
		err = json.Unmarshal([]byte(model.SecretData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.SecretData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		_, err = impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at env level ", "error", err)
			return false, err
		}
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretEnv(model)
		if err != nil {
			return false, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromEnvLevelConfig(model, repository.SECRET_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for secret env history", "err", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)
	}

	return true, nil
}

/////

func (impl ConfigMapServiceImpl) CMGlobalDeleteByAppId(name string, appId int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.ConfigsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		err = impl.scopedVariableManager.CreateVariableMappingsForCMApp(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at app level", "error", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)

	}

	return true, nil
}

func (impl ConfigMapServiceImpl) CMEnvironmentDeleteByAppIdAndEnvId(name string, appId int, envId int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(appId, envId)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.ConfigsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.ConfigMapData) > 0 {
		err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.ConfigMapData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		err = impl.scopedVariableManager.CreateVariableMappingsForCMEnv(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at env level", "error", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)
	}

	return true, nil
}

func (impl ConfigMapServiceImpl) CSGlobalDeleteByAppId(name string, appId int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.SecretsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.SecretData) > 0 {
		err = json.Unmarshal([]byte(model.SecretData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.SecretData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		//VARIABLE_MAPPING_UPDATE
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretApp(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at app level", "error", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)

	}

	return true, nil
}

func (impl ConfigMapServiceImpl) CSEnvironmentDeleteByAppIdAndEnvId(name string, appId int, envId int, userId int32) (bool, error) {

	model, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(appId, envId)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return false, err
	}
	configsList := &bean.SecretsList{}
	found := false
	var configs []*bean.ConfigData
	if len(model.SecretData) > 0 {
		err = json.Unmarshal([]byte(model.SecretData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			found = true
		} else {
			configs = append(configs, item)
		}
	}

	if found {
		configsList.ConfigData = configs
		configDataByte, err := json.Marshal(configsList)
		if err != nil {
			return false, err
		}
		model.SecretData = string(configDataByte)
		model.UpdatedBy = userId
		model.UpdatedOn = time.Now()
		//VARIABLE_MAPPING_UPDATE
		//sl := bean.SecretsList{}
		//data, err := sl.GetTransformedDataForSecretList(model.SecretData, util2.DecodeSecret)
		//if err != nil {
		//	return false, err
		//}
		//err = impl.extractAndMapVariables(data, model.Id, repository5.EntityTypeSecretEnvLevel, model.UpdatedBy)
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretEnv(model)
		if err != nil {
			return false, err
		}
		_, err = impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while updating at env level ", "error", err)
			return false, err
		}
	} else {
		impl.logger.Debugw("no config map found for delete with this name", "name", name)
	}

	return true, nil
}

////

func (impl ConfigMapServiceImpl) CSGlobalFetchForEdit(name string, id int) (*bean.ConfigDataRequest, error) {
	configMapEnvLevel, err := impl.configMapRepository.GetByIdAppLevel(id)
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}

	configsList := &bean.SecretsList{}
	var configs []*bean.ConfigData
	if len(configMapEnvLevel.SecretData) > 0 {
		err = json.Unmarshal([]byte(configMapEnvLevel.SecretData), configsList)
		if err != nil {
			impl.logger.Warnw("error while Unmarshal", "error", err)
		}
	}
	for _, item := range configsList.ConfigData {
		if item.Name == name {
			configs = append(configs, item)
			break
		}
	}

	configDataRequest := &bean.ConfigDataRequest{}
	configDataRequest.Id = configMapEnvLevel.Id
	configDataRequest.AppId = configMapEnvLevel.AppId
	configDataRequest.ConfigData = configs
	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CSEnvironmentFetchForEdit(name string, id int, appId int, envId int) (*bean.ConfigDataRequest, error) {
	configDataRequest := &bean.ConfigDataRequest{}
	configsList := &bean.SecretsList{}
	var configs []*bean.ConfigData
	if id > 0 {
		configMapEnvLevel, err := impl.configMapRepository.GetByIdEnvLevel(id)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		if len(configMapEnvLevel.SecretData) > 0 {
			err = json.Unmarshal([]byte(configMapEnvLevel.SecretData), configsList)
			if err != nil {
				impl.logger.Warnw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range configsList.ConfigData {
			if item.Name == name {

				appLevelConfigMap, err := impl.CmCsConfigGlobalFetchUsingAppId(name, appId, bean.CS)
				if err != nil && err != pg.ErrNoRows {
					impl.logger.Errorw("error in fetching app level config", "appId", appId, "err", err)
					return nil, err
				}
				if len(item.MergeStrategy) == 0 && len(appLevelConfigMap.ConfigData) > 0 {
					item.MergeStrategy = models.MERGE_STRATEGY_REPLACE
				}

				configs = append(configs, item)
				break
			}
		}
	}
	if len(configs) == 0 {
		configMapGlobal, err := impl.configMapRepository.GetByAppIdAppLevel(appId)
		if err != nil && pg.ErrNoRows != err {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		if pg.ErrNoRows == err {
			impl.logger.Warnw("no app level secret found for this request", "appId", appId)
		}
		configMapGlobalList := &bean.SecretsList{}
		if len(configMapGlobal.SecretData) > 0 {
			err = json.Unmarshal([]byte(configMapGlobal.SecretData), configMapGlobalList)
			if err != nil {
				impl.logger.Warnw("error while Unmarshal", "error", err)
			}
		}
		for _, item := range configMapGlobalList.ConfigData {
			if item.Name == name {
				configs = append(configs, item)
				break
			}
		}
	}
	configDataRequest.Id = id
	configDataRequest.AppId = appId
	configDataRequest.EnvironmentId = envId
	configDataRequest.ConfigData = configs
	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) validateConfigData(configData *bean.ConfigData) (bool, error) {
	dataMap := make(map[string]string)
	if configData.Data != nil {
		err := json.Unmarshal(configData.Data, &dataMap)
		if err != nil {
			impl.logger.Errorw("error while Unmarshal", "error", err)
			return false, fmt.Errorf("unmarshal failed for data, please provide valid json")
		}
	}
	re := regexp.MustCompile("[-._a-zA-Z0-9]+") //^[A-ZA-Z0-9_]+$
	for key := range dataMap {
		if !re.MatchString(key) {
			return false, fmt.Errorf("invalid key : %s", key)
		}
	}
	return true, nil
}

func (impl ConfigMapServiceImpl) validateConfigDataForSecretsOnly(configData *bean.ConfigData) (bool, error) {

	// check encoding in base64 for secret data
	if len(configData.Data) > 0 {
		dataMap := make(map[string]string)
		err := json.Unmarshal(configData.Data, &dataMap)
		if err != nil {
			impl.logger.Errorw("error while unmarshalling secret data ", "error", err)
			return false, err
		}
		err = util2.ValidateEncodedDataByDecoding(dataMap)
		if err != nil {
			impl.logger.Errorw("error in decoding secret data", "error", err)
			return false, util.DefaultApiError().WithHttpStatusCode(http.StatusUnprocessableEntity).WithCode(strconv.Itoa(http.StatusUnprocessableEntity)).
				WithUserMessage("error in decoding data, make sure the secret data is encoded properly").
				WithInternalMessage("error in decoding data, make sure the secret data is encoded properly")
		}
	}
	if configData.IsESOExternalSecretType() {
		if !configData.External {
			return false, util.DefaultApiError().WithHttpStatusCode(http.StatusBadRequest).WithCode(strconv.Itoa(http.StatusBadRequest)).
				WithUserMessage(fmt.Sprintf("external flag should be true for '%s' secret type", configData.ExternalSecretType)).
				WithInternalMessage(fmt.Sprintf("external flag should be true for '%s' secret type", configData.ExternalSecretType))
		}
		if configData.ESOSecretData.ESODataFrom == nil && configData.ESOSecretData.ESOData == nil {
			return false, util.DefaultApiError().WithHttpStatusCode(http.StatusBadRequest).WithCode(strconv.Itoa(http.StatusBadRequest)).
				WithUserMessage("both esoSecretData.esoDataFrom and esoSecretData.esoData can't be empty").
				WithInternalMessage("both esoSecretData.esoDataFrom and esoSecretData.esoData can't be empty")
		} else if configData.ESOSecretData.SecretStore == nil && configData.ESOSecretData.SecretStoreRef == nil {
			return false, util.DefaultApiError().WithHttpStatusCode(http.StatusBadRequest).WithCode(strconv.Itoa(http.StatusBadRequest)).
				WithUserMessage("both esoSecretData.secretStore and esoSecretData.secretStoreRef can't be empty").
				WithInternalMessage("both esoSecretData.secretStore and esoSecretData.secretStoreRef can't be empty")
		}
	}
	return true, nil
}

func (impl ConfigMapServiceImpl) updateConfigData(configData *bean.ConfigData, syncRequest *bean.BulkPatchRequest) (*bean.ConfigData, error) {
	dataMap := make(map[string]string)
	var updatedData json.RawMessage
	if configData.Data != nil {
		err := json.Unmarshal(configData.Data, &dataMap)
		if err != nil {
			impl.logger.Errorw("error while Unmarshal", "error", err)
			return configData, fmt.Errorf("unmarshal failed for data")
		}
		if syncRequest.PatchAction == 1 {
			dataMap[syncRequest.Key] = syncRequest.Value
		} else if syncRequest.PatchAction == 2 {
			if _, ok := dataMap[syncRequest.Key]; ok {
				dataMap[syncRequest.Key] = syncRequest.Value
			}
		} else if syncRequest.PatchAction == 3 {
			if _, ok := dataMap[syncRequest.Key]; ok {
				delete(dataMap, syncRequest.Key)
			}
		}
		updatedData, err = json.Marshal(dataMap)
		if err != nil {
			impl.logger.Errorw("error while marshal", "error", err)
			return configData, fmt.Errorf("marshal failed for data")
		}
		configData.Data = updatedData
	} else if syncRequest.PatchAction == 1 {
		err := json.Unmarshal(configData.Data, &dataMap)
		if err != nil {
			impl.logger.Errorw("error while Unmarshal", "error", err)
			return configData, fmt.Errorf("unmarshal failed for data")
		}
		dataMap[syncRequest.Key] = syncRequest.Value
		updatedData, err = json.Marshal(dataMap)
		if err != nil {
			impl.logger.Errorw("error while marshal", "error", err)
			return configData, fmt.Errorf("marshal failed for data")
		}
		configData.Data = updatedData
	}
	return configData, nil
}

func (impl ConfigMapServiceImpl) ConfigSecretGlobalBulkPatch(bulkPatchRequest *bean.BulkPatchRequest) (*bean.BulkPatchRequest, error) {
	_, err := impl.buildBulkPayload(bulkPatchRequest)
	if err != nil {
		impl.logger.Errorw("service err, ConfigSecretGlobalBulkPatch", "err", err, "payload", bulkPatchRequest)
		return nil, fmt.Errorf("")
	}
	if len(bulkPatchRequest.Payload) == 0 {
		return nil, fmt.Errorf("invalid request no payload found for sync")
	}
	for _, payload := range bulkPatchRequest.Payload {
		model, err := impl.configMapRepository.GetByAppIdAppLevel(payload.AppId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		if err == pg.ErrNoRows {
			continue
		}
		if bulkPatchRequest.Type == "CM" {
			configsList := &bean.ConfigsList{}
			var configs []*bean.ConfigData
			if len(model.ConfigMapData) > 0 {
				err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
				if err != nil {
					impl.logger.Warnw("error while Unmarshal", "error", err)
				}
			}
			for _, item := range configsList.ConfigData {
				if item.Name == bulkPatchRequest.Name {
					updatedConfigData, err := impl.updateConfigData(item, bulkPatchRequest)
					if err != nil {
						impl.logger.Warnw("error while updating data", "error", err)
					}
					item.Data = updatedConfigData.Data
				}
				configs = append(configs, item)
			}
			configsList.ConfigData = configs
			configDataByte, err := json.Marshal(configsList)
			if err != nil {
				return nil, err
			}
			model.ConfigMapData = string(configDataByte)
		} else if bulkPatchRequest.Type == "CS" {
			secretsList := &bean.SecretsList{}
			var configs []*bean.ConfigData
			if len(model.SecretData) > 0 {
				err = json.Unmarshal([]byte(model.SecretData), secretsList)
				if err != nil {
					impl.logger.Warnw("error while Unmarshal", "error", err)
				}
			}
			for _, item := range secretsList.ConfigData {
				if item.Name == bulkPatchRequest.Name {
					updatedConfigData, err := impl.updateConfigData(item, bulkPatchRequest)
					if err != nil {
						impl.logger.Warnw("error while updating data", "error", err)
					}
					item.Data = updatedConfigData.Data
				}
				configs = append(configs, item)
			}
			secretsList.ConfigData = configs
			configDataByte, err := json.Marshal(secretsList)
			if err != nil {
				return nil, err
			}
			model.SecretData = string(configDataByte)
		}
		model.UpdatedBy = bulkPatchRequest.UserId
		model.UpdatedOn = time.Now()
		_, err = impl.configMapRepository.UpdateAppLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		//VARIABLE_MAPPING_UPDATE
		err = impl.scopedVariableManager.CreateVariableMappingsForCMApp(model)
		if err != nil {
			return nil, err
		}
		//sl := bean.SecretsList{}
		//data, err := sl.GetTransformedDataForSecretList(model.SecretData, util2.DecodeSecret)
		//if err != nil {
		//	return nil, err
		//}
		//err = impl.extractAndMapVariables(data, model.Id, repository5.EntityTypeSecretAppLevel, model.UpdatedBy)
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretApp(model)
		if err != nil {
			return nil, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromAppLevelConfig(model, repository.CONFIGMAP_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for global CM/CS history in bulk update", "err", err)
			return nil, err
		}
	}
	return bulkPatchRequest, nil
}

func (impl ConfigMapServiceImpl) ConfigSecretEnvironmentBulkPatch(bulkPatchRequest *bean.BulkPatchRequest) (*bean.BulkPatchRequest, error) {
	_, err := impl.buildBulkPayload(bulkPatchRequest)
	if err != nil {
		impl.logger.Errorw("service err, ConfigSecretEnvironmentBulkPatch", "err", err, "payload", bulkPatchRequest)
		return nil, fmt.Errorf("")
	}
	if len(bulkPatchRequest.Payload) == 0 {
		return nil, fmt.Errorf("invalid request no payload found for sync")
	}
	for _, payload := range bulkPatchRequest.Payload {
		if payload.AppId == 0 || payload.EnvId == 0 {
			return nil, fmt.Errorf("invalid request payload not complete for env level patch")
		}
	}
	for _, payload := range bulkPatchRequest.Payload {
		model, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(payload.AppId, payload.EnvId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		if err == pg.ErrNoRows {
			continue
		}
		if bulkPatchRequest.Type == "CM" {
			configsList := &bean.ConfigsList{}
			var configs []*bean.ConfigData
			if len(model.ConfigMapData) > 0 {
				err = json.Unmarshal([]byte(model.ConfigMapData), configsList)
				if err != nil {
					impl.logger.Warnw("error while Unmarshal", "error", err)
				}
			}
			for _, item := range configsList.ConfigData {
				if item.Name == bulkPatchRequest.Name {
					updatedConfigData, err := impl.updateConfigData(item, bulkPatchRequest)
					if err != nil {
						impl.logger.Warnw("error while updating data", "error", err)
					}
					item.Data = updatedConfigData.Data
				}
				configs = append(configs, item)
			}
			configsList.ConfigData = configs
			configDataByte, err := json.Marshal(configsList)
			if err != nil {
				return nil, err
			}
			model.ConfigMapData = string(configDataByte)
		} else if bulkPatchRequest.Type == "CS" {
			secretsList := &bean.SecretsList{}
			var configs []*bean.ConfigData
			if len(model.SecretData) > 0 {
				err = json.Unmarshal([]byte(model.SecretData), secretsList)
				if err != nil {
					impl.logger.Warnw("error while Unmarshal", "error", err)
				}
			}
			for _, item := range secretsList.ConfigData {
				if item.Name == bulkPatchRequest.Name {
					updatedConfigData, err := impl.updateConfigData(item, bulkPatchRequest)
					if err != nil {
						impl.logger.Debugw("error while updating data", "error", err)
					}
					item.Data = updatedConfigData.Data
				}
				configs = append(configs, item)
			}
			secretsList.ConfigData = configs
			configDataByte, err := json.Marshal(secretsList)
			if err != nil {
				return nil, err
			}
			model.SecretData = string(configDataByte)
		}
		model.UpdatedBy = bulkPatchRequest.UserId
		model.UpdatedOn = time.Now()
		_, err = impl.configMapRepository.UpdateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while fetching from db", "error", err)
			return nil, err
		}
		//VARIABLE_MAPPING_UPDATE

		err = impl.scopedVariableManager.CreateVariableMappingsForCMEnv(model)
		if err != nil {
			return nil, err
		}
		//sl := bean.SecretsList{}
		//data, err := sl.GetTransformedDataForSecretList(model.SecretData, util2.DecodeSecret)
		//if err != nil {
		//	return nil, err
		//}
		//err = impl.extractAndMapVariables(data, model.Id, repository5.EntityTypeSecretEnvLevel, model.UpdatedBy)
		err = impl.scopedVariableManager.CreateVariableMappingsForSecretEnv(model)
		if err != nil {
			return nil, err
		}
		err = impl.configMapHistoryService.CreateHistoryFromEnvLevelConfig(model, repository.CONFIGMAP_TYPE)
		if err != nil {
			impl.logger.Errorw("error in creating entry for env CM/CS history in bulk update", "err", err)
			return nil, err
		}
	}
	return bulkPatchRequest, nil
}

func (impl ConfigMapServiceImpl) validateExternalSecretChartCompatibility(appId int, envId int, configData *bean.ConfigData) (bool, error) {

	if configData.ExternalSecret != nil && len(configData.ExternalSecret) > 0 {
		for _, es := range configData.ExternalSecret {
			if len(es.Property) > 0 || es.IsBinary == true {
				chartVersion, err := impl.commonService.FetchLatestChartVersion(appId, envId)
				if err != nil {
					return false, err
				}
				chartMajorVersion, chartMinorVersion, err := util2.ExtractChartVersion(chartVersion)
				if err != nil {
					impl.logger.Errorw("chart version parsing", "err", err)
					return false, err
				}
				if chartMajorVersion <= 3 && chartMinorVersion < 8 {
					return false, fmt.Errorf("this chart version dosent support property and isBinary keys, please upgrade chart: %s", configData.Name)
				}
			}
		}
	}
	return true, nil
}

func (impl ConfigMapServiceImpl) buildBulkPayload(bulkPatchRequest *bean.BulkPatchRequest) (*bean.BulkPatchRequest, error) {
	var payload []*bean.BulkPatchPayload
	if bulkPatchRequest.Filter != nil {
		apps, err := impl.appRepository.FetchAppsByFilterV2(bulkPatchRequest.Filter.AppNameIncludes, bulkPatchRequest.Filter.AppNameExcludes, bulkPatchRequest.Filter.EnvId)
		if err != nil {
			impl.logger.Errorw("chart version parsing", "err", err)
			return bulkPatchRequest, err
		}
		for _, item := range apps {
			if bulkPatchRequest.Global {
				payload = append(payload, &bean.BulkPatchPayload{AppId: item.Id})
			} else {
				payload = append(payload, &bean.BulkPatchPayload{AppId: item.Id, EnvId: bulkPatchRequest.Filter.EnvId})
			}
		}
		bulkPatchRequest.Payload = payload
	} else if bulkPatchRequest.ProjectId > 0 && bulkPatchRequest.Global {
		//backward compatibility
		apps, err := impl.appRepository.FindAppsByTeamId(bulkPatchRequest.ProjectId)
		if err != nil {
			impl.logger.Errorw("service err, buildBulkPayload", "err", err, "payload", bulkPatchRequest)
			return bulkPatchRequest, err
		}
		var payload []*bean.BulkPatchPayload
		for _, app := range apps {
			payload = append(payload, &bean.BulkPatchPayload{AppId: app.Id})
		}
		bulkPatchRequest.Payload = payload
	}
	return bulkPatchRequest, nil
}

func (impl ConfigMapServiceImpl) ConfigSecretEnvironmentCreate(createJobEnvOverrideRequest *bean.CreateJobEnvOverridePayload) (*bean.CreateJobEnvOverridePayload, error) {
	configMap, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(createJobEnvOverrideRequest.AppId, createJobEnvOverrideRequest.EnvId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	if err != nil {
		model := &chartConfig.ConfigMapEnvModel{
			AppId:         createJobEnvOverrideRequest.AppId,
			EnvironmentId: createJobEnvOverrideRequest.EnvId,
			Deleted:       false,
		}
		model.CreatedBy = createJobEnvOverrideRequest.UserId
		model.UpdatedBy = createJobEnvOverrideRequest.UserId
		configMap, err = impl.configMapRepository.CreateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating env level", "error", err)
			return nil, err
		}
		return createJobEnvOverrideRequest, nil

	}
	if configMap.Deleted {
		configMap.Deleted = false
		_, err = impl.configMapRepository.UpdateEnvLevel(configMap)
		if err != nil {
			impl.logger.Errorw("error while creating env level", "error", err)
			return nil, err
		}
		return createJobEnvOverrideRequest, nil
	}
	env, err := impl.environmentRepository.FindById(configMap.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error while fetching environment from db", "error", err)
		return nil, err
	}
	impl.logger.Warnw("Environment override in this environment already exits", "appId", createJobEnvOverrideRequest.AppId, "envId", createJobEnvOverrideRequest.EnvId)
	return nil, errors.New("Environment " + env.Name + " already exists.")

}

func (impl ConfigMapServiceImpl) ConfigSecretEnvironmentDelete(createJobEnvOverrideRequest *bean.CreateJobEnvOverridePayload) (*bean.CreateJobEnvOverridePayload, error) {
	configMap, err := impl.configMapRepository.GetByAppIdAndEnvIdEnvLevel(createJobEnvOverrideRequest.AppId, createJobEnvOverrideRequest.EnvId)
	if pg.ErrNoRows == err {
		impl.logger.Warnw("Environment override in this environment doesn't exits", "appId", createJobEnvOverrideRequest.AppId, "envId", createJobEnvOverrideRequest.EnvId)
		return nil, err
	}
	if err != nil {
		impl.logger.Errorw("error while fetching from db", "error", err)
		return nil, err
	}
	configMap.Deleted = true
	configMap.ConfigMapData = ""
	configMap.SecretData = ""
	configMap.UpdatedBy = createJobEnvOverrideRequest.UserId
	_, err = impl.configMapRepository.UpdateEnvLevel(configMap)
	if err != nil {
		impl.logger.Errorw("error while updating env level", "error", err)
		return nil, err
	}
	return createJobEnvOverrideRequest, nil
}

func (impl ConfigMapServiceImpl) ConfigSecretEnvironmentGet(appId int) ([]bean.JobEnvOverrideResponse, error) {
	configMap, err := impl.configMapRepository.GetEnvLevelByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error while fetching envConfig from db", "error", err)
		return nil, err
	}
	var envIds []*int
	for _, cm := range configMap {
		envIds = append(envIds, &cm.EnvironmentId)
	}
	var jobEnvOverrideResponse []bean.JobEnvOverrideResponse

	if len(envIds) == 0 {
		return jobEnvOverrideResponse, nil
	}
	envs, err := impl.environmentRepository.FindByIds(envIds)

	if err != nil {
		impl.logger.Errorw("error while fetching environments from db", "error", err)
		return nil, err
	}

	envIdNameMap := make(map[int]string)

	for _, env := range envs {

		envIdNameMap[env.Id] = env.Name
	}

	for _, cm := range configMap {
		jobEnvOverride := bean.JobEnvOverrideResponse{
			EnvironmentId:   cm.EnvironmentId,
			AppId:           cm.AppId,
			Id:              cm.Id,
			EnvironmentName: envIdNameMap[cm.EnvironmentId],
		}
		jobEnvOverrideResponse = append(jobEnvOverrideResponse, jobEnvOverride)
	}

	return jobEnvOverrideResponse, nil
}

func (impl ConfigMapServiceImpl) ConfigSecretEnvironmentClone(appId int, cloneAppId int, userId int32) ([]chartConfig.ConfigMapEnvModel, error) {
	configMap, err := impl.configMapRepository.GetEnvLevelByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error while fetching envConfig from db", "error", err)
		return nil, err
	}
	var jobEnvOverrideResponse []chartConfig.ConfigMapEnvModel

	if err != nil {
		impl.logger.Errorw("error while fetching environments from db", "error", err)
		return nil, err
	}

	for _, cm := range configMap {
		model := &chartConfig.ConfigMapEnvModel{
			AppId:         cloneAppId,
			EnvironmentId: cm.EnvironmentId,
			ConfigMapData: cm.ConfigMapData,
			SecretData:    cm.SecretData,
			Deleted:       cm.Deleted,
			AuditLog: sql.AuditLog{
				CreatedBy: userId,
				UpdatedBy: userId,
			},
		}

		_, err := impl.configMapRepository.CreateEnvLevel(model)
		if err != nil {
			impl.logger.Errorw("error while creating env level", "error", err)
			return nil, err
		}
		jobEnvOverrideResponse = append(jobEnvOverrideResponse, *model)
	}

	return jobEnvOverrideResponse, nil
}
func (impl ConfigMapServiceImpl) FetchCmCsNamesAppAndEnvLevel(appId int, envId int) ([]bean.ConfigNameAndType, []bean.ConfigNameAndType, error) {
	var cMCSNamesEnvLevel []bean.ConfigNameAndType

	cMCSNamesAppLevel, err := impl.configMapRepository.GetConfigNamesForAppAndEnvLevel(appId, -1)
	if err != nil {
		impl.logger.Errorw("error in fetching CM/CS names at app level ", "appId", appId, "err", err)
		return nil, nil, err
	}
	if envId > 0 {
		cMCSNamesEnvLevel, err = impl.configMapRepository.GetConfigNamesForAppAndEnvLevel(appId, envId)
		if err != nil {
			impl.logger.Errorw("error in fetching CM/CS names  at env level ", "appId", appId, "envId", envId, "err", err)
			return nil, nil, err
		}
	}
	return cMCSNamesAppLevel, cMCSNamesEnvLevel, nil
}

func (impl ConfigMapServiceImpl) CmCsConfigGlobalFetchUsingAppId(name string, appId int, resourceType bean.ResourceType) (*bean.ConfigDataRequest, error) {
	var fetchGlobalConfigFunc func(int) (*bean.ConfigDataRequest, error)
	if resourceType == bean.CS {
		fetchGlobalConfigFunc = impl.CSGlobalFetch
	} else if resourceType == bean.CM {
		fetchGlobalConfigFunc = impl.CMGlobalFetch
	}
	configDataRequest, err := fetchGlobalConfigFunc(appId)
	if err != nil {
		impl.logger.Errorw("error in fetching global cm using app id ", "cmName", name, "appId", appId, "err", err)
		return nil, err
	}
	configDataRequest.Deletable = true // always true in oss
	configs := make([]*bean.ConfigData, 0, len(configDataRequest.ConfigData))
	for _, configData := range configDataRequest.ConfigData {
		if configData.Name == name {
			configs = append(configs, configData)
		}
	}
	configDataRequest.ConfigData = configs
	return configDataRequest, nil
}

func (impl ConfigMapServiceImpl) CmCsConfigOverrideFetchUsingAppAndEnvId(name string, appId, envId int, resourceType bean.ResourceType) (*bean.ConfigDataRequest, error) {
	var fetchGlobalConfigFunc func(int, int) (*bean.ConfigDataRequest, error)
	if resourceType == bean.CS {
		fetchGlobalConfigFunc = impl.CSEnvironmentFetch
	} else if resourceType == bean.CM {
		fetchGlobalConfigFunc = impl.CMEnvironmentFetch
	}
	configDataRequest, err := fetchGlobalConfigFunc(appId, envId)
	if err != nil {
		impl.logger.Errorw("error in fetching global cm using app id ", "cmName", name, "appId", appId, "err", err)
		return nil, err
	}
	configs := make([]*bean.ConfigData, 0, len(configDataRequest.ConfigData))
	for _, configData := range configDataRequest.ConfigData {
		if configData.Name == name {
			configs = append(configs, configData)
		}
	}
	configDataRequest.ConfigData = configs
	return configDataRequest, nil
}
