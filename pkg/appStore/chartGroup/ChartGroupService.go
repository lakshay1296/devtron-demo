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

package chartGroup

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/devtron-labs/devtron/client/argocdServer"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/timelineStatus"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/app/status"
	"github.com/devtron-labs/devtron/pkg/appStore/adapter"
	appStoreDiscoverRepository "github.com/devtron-labs/devtron/pkg/appStore/discover/repository"
	service2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/service"
	"github.com/devtron-labs/devtron/pkg/appStore/installedApp/service/FullMode"
	"github.com/devtron-labs/devtron/pkg/appStore/installedApp/service/FullMode/deployment"
	"github.com/devtron-labs/devtron/pkg/appStore/values/service"
	bean3 "github.com/devtron-labs/devtron/pkg/cluster/bean"
	cluster2 "github.com/devtron-labs/devtron/pkg/cluster/environment"
	bean2 "github.com/devtron-labs/devtron/pkg/cluster/environment/bean"
	commonBean "github.com/devtron-labs/devtron/pkg/deployment/gitOps/common/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/git"
	"github.com/devtron-labs/devtron/pkg/eventProcessor/out"
	"github.com/devtron-labs/devtron/pkg/team/read"
	repository3 "github.com/devtron-labs/devtron/pkg/team/repository"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/chartGroup/repository"
	"github.com/devtron-labs/devtron/pkg/appStore/installedApp/repository"
	appStoreValuesRepository "github.com/devtron-labs/devtron/pkg/appStore/values/repository"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type ChartGroupServiceImpl struct {
	logger                               *zap.SugaredLogger
	chartGroupEntriesRepository          repository2.ChartGroupEntriesRepository
	chartGroupRepository                 repository2.ChartGroupReposotory
	chartGroupDeploymentRepository       repository2.ChartGroupDeploymentRepository
	installedAppRepository               repository.InstalledAppRepository
	appStoreVersionValuesRepository      appStoreValuesRepository.AppStoreVersionValuesRepository
	appStoreRepository                   appStoreDiscoverRepository.AppStoreRepository
	userAuthService                      user.UserAuthService
	appStoreApplicationVersionRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository
	environmentService                   cluster2.EnvironmentService
	teamRepository                       repository3.TeamRepository
	clusterInstalledAppsRepository       repository.ClusterInstalledAppsRepository
	appStoreValuesService                service.AppStoreValuesService
	appStoreDeploymentService            service2.AppStoreDeploymentService
	appStoreDeploymentDBService          service2.AppStoreDeploymentDBService
	pipelineStatusTimelineService        status.PipelineStatusTimelineService
	acdConfig                            *argocdServer.ACDConfig
	fullModeDeploymentService            deployment.FullModeDeploymentService
	gitOperationService                  git.GitOperationService
	installAppService                    FullMode.InstalledAppDBExtendedService
	appStoreAppsEventPublishService      out.AppStoreAppsEventPublishService
	teamReadService                      read.TeamReadService
}

func NewChartGroupServiceImpl(logger *zap.SugaredLogger,
	chartGroupEntriesRepository repository2.ChartGroupEntriesRepository,
	chartGroupRepository repository2.ChartGroupReposotory,
	chartGroupDeploymentRepository repository2.ChartGroupDeploymentRepository,
	installedAppRepository repository.InstalledAppRepository,
	appStoreVersionValuesRepository appStoreValuesRepository.AppStoreVersionValuesRepository,
	appStoreRepository appStoreDiscoverRepository.AppStoreRepository,
	userAuthService user.UserAuthService,
	appStoreApplicationVersionRepository appStoreDiscoverRepository.AppStoreApplicationVersionRepository,
	environmentService cluster2.EnvironmentService,
	teamRepository repository3.TeamRepository,
	clusterInstalledAppsRepository repository.ClusterInstalledAppsRepository,
	appStoreValuesService service.AppStoreValuesService,
	appStoreDeploymentService service2.AppStoreDeploymentService,
	appStoreDeploymentDBService service2.AppStoreDeploymentDBService,
	pipelineStatusTimelineService status.PipelineStatusTimelineService,
	acdConfig *argocdServer.ACDConfig,
	fullModeDeploymentService deployment.FullModeDeploymentService,
	gitOperationService git.GitOperationService,
	installAppService FullMode.InstalledAppDBExtendedService,
	appStoreAppsEventPublishService out.AppStoreAppsEventPublishService,
	teamReadService read.TeamReadService) (*ChartGroupServiceImpl, error) {
	impl := &ChartGroupServiceImpl{
		logger:                               logger,
		chartGroupEntriesRepository:          chartGroupEntriesRepository,
		chartGroupRepository:                 chartGroupRepository,
		chartGroupDeploymentRepository:       chartGroupDeploymentRepository,
		installedAppRepository:               installedAppRepository,
		appStoreVersionValuesRepository:      appStoreVersionValuesRepository,
		userAuthService:                      userAuthService,
		appStoreApplicationVersionRepository: appStoreApplicationVersionRepository,
		environmentService:                   environmentService,
		teamRepository:                       teamRepository,
		clusterInstalledAppsRepository:       clusterInstalledAppsRepository,
		appStoreValuesService:                appStoreValuesService,
		appStoreDeploymentService:            appStoreDeploymentService,
		appStoreDeploymentDBService:          appStoreDeploymentDBService,
		pipelineStatusTimelineService:        pipelineStatusTimelineService,
		acdConfig:                            acdConfig,
		fullModeDeploymentService:            fullModeDeploymentService,
		gitOperationService:                  gitOperationService,
		installAppService:                    installAppService,
		appStoreAppsEventPublishService:      appStoreAppsEventPublishService,
		appStoreRepository:                   appStoreRepository,
		teamReadService:                      teamReadService,
	}
	return impl, nil
}

type ChartGroupService interface {
	CreateChartGroup(req *ChartGroupBean) (*ChartGroupBean, error)
	UpdateChartGroup(req *ChartGroupBean) (*ChartGroupBean, error)
	SaveChartGroupEntries(req *ChartGroupBean) (*ChartGroupBean, error)
	GetChartGroupWithChartMetaData(chartGroupId int) (*ChartGroupBean, error)
	GetChartGroupList(max int) (*ChartGroupList, error)
	GetChartGroupWithInstallationDetail(chartGroupId int) (*ChartGroupBean, error)
	ChartGroupListMin(max int) ([]*ChartGroupBean, error)
	DeleteChartGroup(req *ChartGroupBean) error

	DeployBulk(chartGroupInstallRequest *ChartGroupInstallRequest) (*ChartGroupInstallAppRes, error)
	DeployDefaultChartOnCluster(bean *bean3.ClusterBean, userId int32) (bool, error)
	TriggerDeploymentEventAndHandleStatusUpdate(installAppVersions []*appStoreBean.InstallAppVersionDTO)

	PerformDeployStage(installedAppVersionId int, installedAppVersionHistoryId int, userId int32) (*appStoreBean.InstallAppVersionDTO, error)
}

type ChartGroupList struct {
	Groups []*ChartGroupBean `json:"groups,omitempty"`
}
type ChartGroupBean struct {
	Name               string                 `json:"name,omitempty" validate:"name-component,max=200,min=5"`
	Description        string                 `json:"description,omitempty"`
	Id                 int                    `json:"id,omitempty"`
	ChartGroupEntries  []*ChartGroupEntryBean `json:"chartGroupEntries,omitempty"`
	InstalledChartData []*InstalledChartData  `json:"installedChartData,omitempty"`
	UserId             int32                  `json:"-"`
}

type ChartGroupEntryBean struct {
	Id                           int            `json:"id,omitempty"`
	AppStoreValuesVersionId      int            `json:"appStoreValuesVersionId,omitempty"` //AppStoreVersionValuesId
	AppStoreValuesVersionName    string         `json:"appStoreValuesVersionName,omitempty"`
	AppStoreValuesChartVersion   string         `json:"appStoreValuesChartVersion,omitempty"`   //chart version corresponding to values
	AppStoreApplicationVersionId int            `json:"appStoreApplicationVersionId,omitempty"` //AppStoreApplicationVersionId
	ChartMetaData                *ChartMetaData `json:"chartMetaData,omitempty"`
	ReferenceType                string         `json:"referenceType, omitempty"`
}

type ChartMetaData struct {
	ChartName                  string `json:"chartName,omitempty"`
	ChartRepoName              string `json:"chartRepoName,omitempty"`
	Icon                       string `json:"icon,omitempty"`
	AppStoreId                 int    `json:"appStoreId"`
	AppStoreApplicationVersion string `json:"appStoreApplicationVersion"`
	EnvironmentName            string `json:"environmentName,omitempty"`
	EnvironmentId              int    `json:"environmentId,omitempty"` //FIXME REMOVE THIS ATTRIBUTE AFTER REMOVING ENVORONMENTID FROM GETINSTALLEDAPPCALL
	IsChartRepoActive          bool   `json:"isChartRepoActive"`
}

type InstalledChartData struct {
	InstallationTime time.Time         `json:"installationTime,omitempty"`
	InstalledCharts  []*InstalledChart `json:"installedCharts,omitempty"`
}

type InstalledChart struct {
	ChartMetaData
	InstalledAppId int `json:"installedAppId,omitempty"`
}

const AppNameAlreadyExistsError = "A chart with this name already exist"

func (impl *ChartGroupServiceImpl) CreateChartGroup(req *ChartGroupBean) (*ChartGroupBean, error) {
	impl.logger.Debugw("chart group create request", "req", req)

	exist, err := impl.chartGroupRepository.FindByName(req.Name)
	if err != nil {
		impl.logger.Errorw("error in creating chart group", "req", req, "err", err)
		return nil, err
	}
	if exist {
		impl.logger.Errorw("Chart with this name already exist", "req", req, "err", err)
		return nil, errors.New(AppNameAlreadyExistsError)
	}

	chartGrouModel := &repository2.ChartGroup{
		Name:        req.Name,
		Description: req.Description,
		AuditLog: sql.AuditLog{
			CreatedOn: time.Now(),
			CreatedBy: req.UserId,
			UpdatedOn: time.Now(),
			UpdatedBy: req.UserId,
		},
	}
	group, err := impl.chartGroupRepository.Save(chartGrouModel)
	if err != nil {
		impl.logger.Errorw("error in creating chart group", "req", chartGrouModel, "err", err)
		return nil, err
	}
	req.Id = group.Id
	return req, nil
}

func (impl *ChartGroupServiceImpl) UpdateChartGroup(req *ChartGroupBean) (*ChartGroupBean, error) {
	impl.logger.Debugw("chart group update request", "req", req)
	chartGrouModel := &repository2.ChartGroup{
		Name:        req.Name,
		Description: req.Description,
		Id:          req.Id,
		AuditLog: sql.AuditLog{
			UpdatedOn: time.Now(),
			UpdatedBy: req.UserId,
		},
	}
	group, err := impl.chartGroupRepository.Update(chartGrouModel)

	if err != nil {
		impl.logger.Errorw("error in update chart group", "req", chartGrouModel, "err", err)
		return nil, err
	}
	req.Id = group.Id
	return req, nil
}

func (impl *ChartGroupServiceImpl) SaveChartGroupEntries(req *ChartGroupBean) (*ChartGroupBean, error) {
	group, err := impl.chartGroupRepository.FindByIdWithEntries(req.Id)
	if err != nil {
		impl.logger.Errorw("error in fetching chart group", "id", req.Id, "err", err)
		return nil, err
	}
	var newEntries []*ChartGroupEntryBean
	oldEntriesMap := make(map[int]*ChartGroupEntryBean)
	for _, entryBean := range req.ChartGroupEntries {
		if entryBean.Id != 0 {
			oldEntriesMap[entryBean.Id] = entryBean
			//update
		} else {
			//create
			newEntries = append(newEntries, entryBean)
		}
	}
	var updateEntries []*repository2.ChartGroupEntry
	for _, existingEntry := range group.ChartGroupEntries {
		if entry, ok := oldEntriesMap[existingEntry.Id]; ok {
			//update
			existingEntry.AppStoreApplicationVersionId = entry.AppStoreApplicationVersionId
			existingEntry.AppStoreValuesVersionId = entry.AppStoreValuesVersionId
		} else {
			//delete
			existingEntry.Deleted = true
		}
		existingEntry.UpdatedBy = req.UserId
		existingEntry.UpdatedOn = time.Now()
		updateEntries = append(updateEntries, existingEntry)
	}

	var createEntries []*repository2.ChartGroupEntry
	for _, entryBean := range newEntries {
		entry := &repository2.ChartGroupEntry{
			AppStoreValuesVersionId:      entryBean.AppStoreValuesVersionId,
			AppStoreApplicationVersionId: entryBean.AppStoreApplicationVersionId,
			ChartGroupId:                 group.Id,
			Deleted:                      false,
			AuditLog: sql.AuditLog{
				CreatedOn: time.Now(),
				CreatedBy: req.UserId,
				UpdatedOn: time.Now(),
				UpdatedBy: req.UserId,
			},
		}
		createEntries = append(createEntries, entry)
	}
	finalEntries, err := impl.chartGroupEntriesRepository.SaveAndUpdateInTransaction(createEntries, updateEntries)
	if err != nil {
		impl.logger.Errorw("error in adding entries", "err", err)
		return nil, err
	}
	impl.logger.Debugw("all entries,", "entry", finalEntries)
	return impl.GetChartGroupWithChartMetaData(req.Id)
}

func (impl *ChartGroupServiceImpl) GetChartGroupWithChartMetaData(chartGroupId int) (*ChartGroupBean, error) {
	chartGroup, err := impl.chartGroupRepository.FindById(chartGroupId)
	if err != nil {
		return nil, err
	}
	chartGroupEntries, err := impl.chartGroupEntriesRepository.FindEntriesWithChartMetaByChartGroupId([]int{chartGroupId})
	if err != nil {
		return nil, err
	}
	chartGroupRes := &ChartGroupBean{
		Name:        chartGroup.Name,
		Description: chartGroup.Description,
		Id:          chartGroup.Id,
	}
	for _, chartGroupEntry := range chartGroupEntries {
		entry := impl.charterEntryAdopter(chartGroupEntry)
		chartGroupRes.ChartGroupEntries = append(chartGroupRes.ChartGroupEntries, entry)
	}
	return chartGroupRes, err
}

func (impl *ChartGroupServiceImpl) charterEntryAdopter(chartGroupEntry *repository2.ChartGroupEntry) *ChartGroupEntryBean {

	var referenceType string
	var valueVersionName string
	var appStoreValuesChartVersion string
	if chartGroupEntry.AppStoreValuesVersionId == 0 {
		referenceType = appStoreBean.REFERENCE_TYPE_DEFAULT
		appStoreValuesChartVersion = chartGroupEntry.AppStoreApplicationVersion.Version
	} else {
		referenceType = appStoreBean.REFERENCE_TYPE_TEMPLATE
		valueVersionName = chartGroupEntry.AppStoreValuesVersion.Name
		//FIXME: orm join not working.  to quick fix it
		valuesVersion, err := impl.appStoreVersionValuesRepository.FindById(chartGroupEntry.AppStoreValuesVersionId)
		if err != nil {
			return nil
		} else {
			appStoreValuesChartVersion = valuesVersion.AppStoreApplicationVersion.Version
		}

		//appStoreValuesChartVersion = chartGroupEntry.AppStoreValuesVersion.AppStoreApplicationVersion.Version
	}
	var chartRepoName string
	var isChartRepoActive bool

	if chartGroupEntry.AppStoreApplicationVersion.AppStore.DockerArtifactStore != nil {
		chartRepoName = chartGroupEntry.AppStoreApplicationVersion.AppStore.DockerArtifactStore.Id
		isChartRepoActive = chartGroupEntry.AppStoreApplicationVersion.AppStore.DockerArtifactStore.OCIRegistryConfig[0].IsChartPullActive
	} else {
		chartRepoName = chartGroupEntry.AppStoreApplicationVersion.AppStore.ChartRepo.Name
		isChartRepoActive = chartGroupEntry.AppStoreApplicationVersion.AppStore.ChartRepo.Active
	}
	entry := &ChartGroupEntryBean{
		Id:                           chartGroupEntry.Id,
		AppStoreValuesVersionId:      chartGroupEntry.AppStoreValuesVersionId,
		AppStoreApplicationVersionId: chartGroupEntry.AppStoreApplicationVersionId,
		ReferenceType:                referenceType,
		AppStoreValuesVersionName:    valueVersionName,
		AppStoreValuesChartVersion:   appStoreValuesChartVersion,
		ChartMetaData: &ChartMetaData{
			ChartName:                  chartGroupEntry.AppStoreApplicationVersion.Name,
			ChartRepoName:              chartRepoName,
			Icon:                       chartGroupEntry.AppStoreApplicationVersion.Icon,
			AppStoreId:                 chartGroupEntry.AppStoreApplicationVersion.AppStoreId,
			AppStoreApplicationVersion: chartGroupEntry.AppStoreApplicationVersion.Version,
			IsChartRepoActive:          isChartRepoActive,
		},
	}
	return entry
}

func (impl *ChartGroupServiceImpl) GetChartGroupList(max int) (*ChartGroupList, error) {
	groups, err := impl.chartGroupRepository.GetAll(max)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}
	var groupIds []int
	groupMap := make(map[int]*ChartGroupBean)
	for _, group := range groups {
		chartGroupRes := &ChartGroupBean{
			Name:        group.Name,
			Description: group.Description,
			Id:          group.Id,
		}
		groupMap[group.Id] = chartGroupRes
		groupIds = append(groupIds, group.Id)
	}
	groupEntries, err := impl.chartGroupEntriesRepository.FindEntriesWithChartMetaByChartGroupId(groupIds)
	if err != nil {
		return nil, err
	}
	for _, groupentry := range groupEntries {
		entry := impl.charterEntryAdopter(groupentry)
		entries := groupMap[groupentry.ChartGroupId].ChartGroupEntries
		entries = append(entries, entry)
		groupMap[groupentry.ChartGroupId].ChartGroupEntries = entries
	}
	var chartGroups []*ChartGroupBean
	for _, v := range groupMap {
		chartGroups = append(chartGroups, v)
	}
	if len(chartGroups) == 0 {
		chartGroups = make([]*ChartGroupBean, 0)
	}
	return &ChartGroupList{Groups: chartGroups}, nil
}

func (impl *ChartGroupServiceImpl) GetChartGroupWithInstallationDetail(chartGroupId int) (*ChartGroupBean, error) {
	chartGroupBean, err := impl.GetChartGroupWithChartMetaData(chartGroupId)
	if err != nil {
		return nil, err
	}
	deployments, err := impl.chartGroupDeploymentRepository.FindByChartGroupId(chartGroupId)
	if err != nil {
		impl.logger.Errorw("error in finding deployment", "chartGroupId", chartGroupId, "err", err)
		return nil, err
	}
	groupDeploymentMap := make(map[string][]*repository2.ChartGroupDeployment)
	for _, deployment := range deployments {
		groupDeploymentMap[deployment.GroupInstallationId] = append(groupDeploymentMap[deployment.GroupInstallationId], deployment)
	}
	for _, groupDeployments := range groupDeploymentMap {
		installedChartData := &InstalledChartData{}
		//installedChartData.InstallationTime
		for _, deployment := range groupDeployments {
			installedChartData.InstallationTime = deployment.CreatedOn
			versions, err := impl.installedAppRepository.GetInstalledAppVersionByInstalledAppIdMeta(deployment.InstalledAppId)
			if err != nil {
				return nil, err
			}
			version := versions[0]
			var chartRepoName string
			var isChartRepoActive bool
			if version.AppStoreApplicationVersion.AppStore.DockerArtifactStore != nil {
				chartRepoName = version.AppStoreApplicationVersion.AppStore.DockerArtifactStore.Id
				isChartRepoActive = version.AppStoreApplicationVersion.AppStore.DockerArtifactStore.OCIRegistryConfig[0].IsChartPullActive
			} else {
				chartRepoName = version.AppStoreApplicationVersion.AppStore.ChartRepo.Name
				isChartRepoActive = version.AppStoreApplicationVersion.AppStore.ChartRepo.Active
			}
			installedChart := &InstalledChart{
				ChartMetaData: ChartMetaData{
					ChartName:         version.InstalledApp.App.AppName,
					ChartRepoName:     chartRepoName,
					Icon:              version.AppStoreApplicationVersion.Icon,
					AppStoreId:        version.AppStoreApplicationVersion.AppStoreId,
					EnvironmentName:   version.InstalledApp.Environment.Name,
					EnvironmentId:     version.InstalledApp.EnvironmentId,
					IsChartRepoActive: isChartRepoActive,
				},
				InstalledAppId: version.InstalledAppId,
			}
			installedChartData.InstalledCharts = append(installedChartData.InstalledCharts, installedChart)
		}

		chartGroupBean.InstalledChartData = append(chartGroupBean.InstalledChartData, installedChartData)
	}
	/*	for _, deployment := range deployments {
		versions, err := impl.installedAppRepository.GetInstalledAppVersionByInstalledAppIdMeta(deployment.InstalledAppId)
		if err != nil {
			return nil, err
		}
		version := versions[0]
		installedChartData := &InstalledChartData{
			InstallationTime: version.CreatedOn,
			InstalledCharts: []*InstalledChart{&InstalledChart{
				ChartMetaData: ChartMetaData{
					ChartName:       version.InstalledApp.App.AppName,
					ChartRepoName:   version.AppStoreApplicationVersion.AppStore.ChartRepo.Name,
					Icon:            version.AppStoreApplicationVersion.Icon,
					AppStoreId:      version.AppStoreApplicationVersion.AppStoreId,
					EnvironmentName: version.InstalledApp.Environment.Name,
					EnvironmentId:   version.InstalledApp.EnvironmentId,
				},
				InstalledAppId: version.InstalledAppId,
			}},
		}
		chartGroupBean.InstalledChartData = append(chartGroupBean.InstalledChartData, installedChartData)
	}*/
	return chartGroupBean, nil
}

func (impl *ChartGroupServiceImpl) ChartGroupListMin(max int) ([]*ChartGroupBean, error) {
	var chartGroupList []*ChartGroupBean
	groups, err := impl.chartGroupRepository.GetAll(max)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		chartGroupRes := &ChartGroupBean{
			Name: group.Name,
			Id:   group.Id,
		}
		chartGroupList = append(chartGroupList, chartGroupRes)
	}
	if len(chartGroupList) == 0 {
		chartGroupList = make([]*ChartGroupBean, 0)
	}
	return chartGroupList, nil
}

func (impl *ChartGroupServiceImpl) DeleteChartGroup(req *ChartGroupBean) error {
	//finding existing
	existingChartGroup, err := impl.chartGroupRepository.FindById(req.Id)
	if err != nil {
		impl.logger.Errorw("No matching entry found for delete.", "err", err, "id", req.Id)
		return err
	}
	//finding chart mappings by group id
	chartGroupMappings, err := impl.chartGroupEntriesRepository.FindEntriesWithChartMetaByChartGroupId([]int{req.Id})
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting chart group entries, DeleteChartGroup", "err", err, "chartGroupId", req.Id)
		return err
	}
	var chartGroupMappingIds []int
	for _, chartGroupMapping := range chartGroupMappings {
		chartGroupMappingIds = append(chartGroupMappingIds, chartGroupMapping.Id)
	}
	dbConnection := impl.installedAppRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		impl.logger.Errorw("error in establishing connection", "err", err)
		return err
	}
	// Rollback tx on error.
	defer tx.Rollback()

	//deleting chart mapping in group
	if len(chartGroupMappingIds) > 0 {
		_, err = impl.chartGroupEntriesRepository.MarkChartGroupEntriesDeleted(chartGroupMappingIds, tx)
		if err != nil {
			impl.logger.Errorw("error in deleting chart group mappings", "err", err)
			return err
		}
	}
	//deleting chart group
	err = impl.chartGroupRepository.MarkChartGroupDeleted(existingChartGroup.Id, tx)
	if err != nil {
		impl.logger.Errorw("error in deleting chart group", "err", err, "chartGroupId", existingChartGroup.Id)
		return err
	}
	//deleting auth roles entries for this chart group
	err = impl.userAuthService.DeleteRoles(bean.CHART_GROUP_TYPE, req.Name, tx, "", "")
	if err != nil {
		impl.logger.Errorw("error in deleting auth roles", "err", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (impl *ChartGroupServiceImpl) DeployBulk(chartGroupInstallRequest *ChartGroupInstallRequest) (*ChartGroupInstallAppRes, error) {
	impl.logger.Debugw("bulk app install request", "req", chartGroupInstallRequest)
	//save in db
	// raise nats event

	var installAppVersionDTOList []*appStoreBean.InstallAppVersionDTO
	for _, chartGroupInstall := range chartGroupInstallRequest.ChartGroupInstallChartRequest {
		installAppVersionDTO, err := impl.requestBuilderForBulkDeployment(chartGroupInstall, chartGroupInstallRequest.ProjectId, chartGroupInstallRequest.UserId)
		if err != nil {
			impl.logger.Errorw("DeployBulk, error in request builder", "err", err)
			return nil, err
		}
		installAppVersionDTOList = append(installAppVersionDTOList, installAppVersionDTO)
	}
	dbConnection := impl.installedAppRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return nil, err
	}
	var installAppVersions []*appStoreBean.InstallAppVersionDTO
	// Rollback tx on error.
	defer tx.Rollback()
	for _, installAppVersionDTO := range installAppVersionDTOList {
		installAppVersionDTO, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationDB(installAppVersionDTO, tx, appStoreBean.BULK_DEPLOY_REQUEST)
		if err != nil {
			impl.logger.Errorw("DeployBulk, error while app store deploy db operation", "err", err)
			return nil, err
		}
		installAppVersions = append(installAppVersions, installAppVersionDTO)
	}
	if chartGroupInstallRequest.ChartGroupId > 0 {
		groupINstallationId, err := getInstallationId(installAppVersions)
		if err != nil {
			return nil, err
		}
		for _, installAppVersionDTO := range installAppVersions {
			chartGroupEntry := createChartGroupEntryObject(installAppVersionDTO, chartGroupInstallRequest.ChartGroupId, groupINstallationId)
			err := impl.chartGroupDeploymentRepository.Save(tx, chartGroupEntry)
			if err != nil {
				impl.logger.Errorw("DeployBulk, error in creating ChartGroupEntryObject", "err", err)
				return nil, err
			}
		}
	}
	//commit transaction
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("DeployBulk, error in tx commit", "err", err)
		return nil, err
	}
	//nats event
	impl.TriggerDeploymentEventAndHandleStatusUpdate(installAppVersions)
	// TODO refactoring: why empty obj ??
	return &ChartGroupInstallAppRes{}, nil
}

func (impl *ChartGroupServiceImpl) requestBuilderForBulkDeployment(installRequest *ChartGroupInstallChartRequest, projectId int, userId int32) (*appStoreBean.InstallAppVersionDTO, error) {
	valYaml := installRequest.ValuesOverrideYaml
	if valYaml == "" {
		valVersion, err := impl.appStoreValuesService.FindValuesByIdAndKind(installRequest.ReferenceValueId, installRequest.ReferenceValueKind)
		if err != nil {
			return nil, err
		}
		valYaml = valVersion.Values
	}
	req := &appStoreBean.InstallAppVersionDTO{
		AppName:            installRequest.AppName,
		TeamId:             projectId,
		EnvironmentId:      installRequest.EnvironmentId,
		AppStoreVersion:    installRequest.AppStoreVersion,
		ValuesOverrideYaml: valYaml,
		UserId:             userId,
		ReferenceValueId:   installRequest.ReferenceValueId,
		ReferenceValueKind: installRequest.ReferenceValueKind,
		ChartGroupEntryId:  installRequest.ChartGroupEntryId,
	}
	return req, nil
}

// generate unique installation ID using APPID
func getInstallationId(installAppVersions []*appStoreBean.InstallAppVersionDTO) (string, error) {
	var buffer bytes.Buffer
	for _, installAppVersionDTO := range installAppVersions {
		if installAppVersionDTO.AppId == 0 {
			return "", fmt.Errorf("app ID not present")
		}
		buffer.WriteString(
			strconv.Itoa(installAppVersionDTO.AppId))
	}
	/* #nosec */
	h := sha1.New()
	_, err := h.Write([]byte(buffer.String()))
	if err != nil {
		return "", err
	}
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}

func createChartGroupEntryObject(installAppVersionDTO *appStoreBean.InstallAppVersionDTO, chartGroupId int, groupINstallationId string) *repository2.ChartGroupDeployment {
	return &repository2.ChartGroupDeployment{
		ChartGroupId:        chartGroupId,
		ChartGroupEntryId:   installAppVersionDTO.ChartGroupEntryId,
		InstalledAppId:      installAppVersionDTO.InstalledAppId,
		Deleted:             false,
		GroupInstallationId: groupINstallationId,
		AuditLog: sql.AuditLog{
			CreatedOn: time.Now(),
			CreatedBy: installAppVersionDTO.UserId,
			UpdatedOn: time.Now(),
			UpdatedBy: installAppVersionDTO.UserId,
		},
	}
}

func (impl *ChartGroupServiceImpl) TriggerDeploymentEventAndHandleStatusUpdate(installAppVersions []*appStoreBean.InstallAppVersionDTO) {
	publishErrMap := impl.appStoreAppsEventPublishService.PublishBulkDeployEvent(installAppVersions)
	for _, version := range installAppVersions {
		var installedAppDeploymentStatus appStoreBean.AppstoreDeploymentStatus
		publishErr, ok := publishErrMap[version.InstalledAppVersionId]
		if !ok || publishErr != nil {
			installedAppDeploymentStatus = appStoreBean.QUE_ERROR
		} else {
			installedAppDeploymentStatus = appStoreBean.ENQUEUED
		}
		if version.Status == appStoreBean.DEPLOY_INIT || version.Status == appStoreBean.QUE_ERROR || version.Status == appStoreBean.ENQUEUED {
			impl.logger.Debugw("status for bulk app-store deploy", "status", installedAppDeploymentStatus)
			_, err := impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(version.InstalledAppVersionId, installedAppDeploymentStatus)
			if err != nil {
				impl.logger.Errorw("error while bulk app-store deploy status update", "err", err)
			}
		}
	}
}

func (impl *ChartGroupServiceImpl) DeployDefaultChartOnCluster(bean *bean3.ClusterBean, userId int32) (bool, error) {
	// STEP 1 - create environment with name "devton"
	impl.logger.Infow("STEP 1", "create environment for cluster component", "clusterId", bean.Id)
	envName := fmt.Sprintf("%d-%s", bean.Id, appStoreBean.DEFAULT_ENVIRONMENT_OR_NAMESPACE_OR_PROJECT)
	env, err := impl.environmentService.FindOne(envName)
	if err != nil && err != pg.ErrNoRows {
		return false, err
	}
	if err == pg.ErrNoRows {
		env = &bean2.EnvironmentBean{
			Environment: envName,
			ClusterId:   bean.Id,
			Namespace:   envName,
			Default:     false,
			Active:      true,
		}
		_, err := impl.environmentService.Create(env, userId)
		if err != nil {
			impl.logger.Errorw("DeployDefaultChartOnCluster, error in creating environment", "data", env, "err", err)
			return false, err
		}
	}

	// STEP 2 - create project with name "devtron"
	impl.logger.Info("STEP 2", "create project for cluster components")
	t, err := impl.teamReadService.FindByTeamName(appStoreBean.DEFAULT_ENVIRONMENT_OR_NAMESPACE_OR_PROJECT)
	if err != nil && err != pg.ErrNoRows {
		return false, err
	}
	if err == pg.ErrNoRows {
		t := &repository3.Team{
			Name:     appStoreBean.DEFAULT_ENVIRONMENT_OR_NAMESPACE_OR_PROJECT,
			Active:   true,
			AuditLog: sql.AuditLog{CreatedBy: userId, CreatedOn: time.Now(), UpdatedOn: time.Now(), UpdatedBy: userId},
		}
		err = impl.teamRepository.Save(t)
		if err != nil {
			impl.logger.Errorw("DeployDefaultChartOnCluster, error in creating team", "data", t, "err", err)
			return false, err
		}
	}

	// STEP 3- read the input data from env variables
	impl.logger.Info("STEP 3", "read the input data from env variables")
	charts := &appStoreBean.ChartComponents{}
	var chartComponents []*appStoreBean.ChartComponent
	if _, err := os.Stat(appStoreBean.CLUSTER_COMPONENT_DIR_PATH); os.IsNotExist(err) {
		impl.logger.Infow("default cluster component directory error", "cluster", bean.ClusterName, "err", err)
		return false, nil
	} else {
		fileInfo, err := ioutil.ReadDir(appStoreBean.CLUSTER_COMPONENT_DIR_PATH)
		if err != nil {
			impl.logger.Errorw("DeployDefaultChartOnCluster, err while reading directory", "err", err)
			return false, err
		}
		for _, file := range fileInfo {
			impl.logger.Infow("file", "name", file.Name())
			if strings.Contains(file.Name(), ".yaml") {
				content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", appStoreBean.CLUSTER_COMPONENT_DIR_PATH, file.Name()))
				if err != nil {
					impl.logger.Errorw("DeployDefaultChartOnCluster, error on reading file", "err", err)
					return false, err
				}
				chartComponent := &appStoreBean.ChartComponent{
					Name:   strings.ReplaceAll(file.Name(), ".yaml", ""),
					Values: string(content),
				}
				chartComponents = append(chartComponents, chartComponent)
			}
		}

		if len(chartComponents) > 0 {
			charts.ChartComponent = chartComponents
			impl.logger.Info("STEP 4 - prepare a bulk request")
			// STEP 4 - prepare a bulk request (unique names need to apply for deploying chart)
			// STEP 4.1 - fetch chart for required name(actual chart name (app-store)) with default values
			// STEP 4.2 - update all the required charts, override values.yaml with env variables.
			chartGroupInstallRequest := &ChartGroupInstallRequest{}
			chartGroupInstallRequest.ProjectId = t.Id
			chartGroupInstallRequest.UserId = userId
			var chartGroupInstallChartRequests []*ChartGroupInstallChartRequest
			for _, item := range charts.ChartComponent {
				appStore, err := impl.appStoreRepository.FindAppStoreByName(item.Name)
				if err != nil {
					impl.logger.Errorw("error in getting app store by name", "appStoreName", item.Name, "err", err)
					return false, err
				}
				isOCIRepo := len(appStore.DockerArtifactStoreId) > 0
				var appStoreApplicationVersionId int
				if isOCIRepo {
					appStoreApplicationVersionId, err = impl.appStoreApplicationVersionRepository.FindLatestVersionByAppStoreIdForOCIRepo(appStore.Id)
					if err != nil {
						impl.logger.Errorw("DeployDefaultChartOnCluster, error in getting app store", "data", t, "err", err)
						return false, err
					}
				} else {
					appStoreApplicationVersionId, err = impl.appStoreApplicationVersionRepository.FindLatestVersionByAppStoreIdForChartRepo(appStore.Id)
					if err != nil {
						impl.logger.Errorw("DeployDefaultChartOnCluster, error in getting app store", "data", t, "err", err)
						return false, err
					}
				}
				chartGroupInstallChartRequest := &ChartGroupInstallChartRequest{
					AppName:            fmt.Sprintf("%d-%d-%s", bean.Id, env.Id, item.Name),
					EnvironmentId:      env.Id,
					ValuesOverrideYaml: item.Values,
					AppStoreVersion:    appStoreApplicationVersionId,
					ReferenceValueId:   appStoreApplicationVersionId,
					ReferenceValueKind: appStoreBean.REFERENCE_TYPE_DEFAULT,
				}
				chartGroupInstallChartRequests = append(chartGroupInstallChartRequests, chartGroupInstallChartRequest)
			}
			chartGroupInstallRequest.ChartGroupInstallChartRequest = chartGroupInstallChartRequests

			impl.logger.Info("STEP 5 - deploy bulk initiated")
			// STEP 5 - deploy
			_, err = impl.deployDefaultComponent(chartGroupInstallRequest)
			if err != nil {
				impl.logger.Errorw("DeployDefaultChartOnCluster, error on bulk deploy", "err", err)
				return false, err
			}
		}
	}
	return true, nil
}

func (impl *ChartGroupServiceImpl) deployDefaultComponent(chartGroupInstallRequest *ChartGroupInstallRequest) (*ChartGroupInstallAppRes, error) {
	impl.logger.Debugw("bulk app install request", "req", chartGroupInstallRequest)
	//save in db
	// raise nats event

	var installAppVersionDTOList []*appStoreBean.InstallAppVersionDTO
	for _, installRequest := range chartGroupInstallRequest.ChartGroupInstallChartRequest {
		installAppVersionDTO, err := impl.requestBuilderForBulkDeployment(installRequest, chartGroupInstallRequest.ProjectId, chartGroupInstallRequest.UserId)
		if err != nil {
			impl.logger.Errorw("DeployBulk, error in request builder", "err", err)
			return nil, err
		}
		installAppVersionDTOList = append(installAppVersionDTOList, installAppVersionDTO)
	}
	dbConnection := impl.installedAppRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return nil, err
	}
	var installAppVersions []*appStoreBean.InstallAppVersionDTO
	// Rollback tx on error.
	defer tx.Rollback()
	for _, installAppVersionDTO := range installAppVersionDTOList {
		installAppVersionDTO, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationDB(installAppVersionDTO, tx, appStoreBean.DEFAULT_COMPONENT_DEPLOYMENT_REQUEST)
		if err != nil {
			impl.logger.Errorw("DeployBulk, error while app store deploy db operation", "err", err)
			return nil, err
		}
		// save installed_app and cluster_id mapping for default chart installation requests
		clusterInstalledAppsModel := adapter.NewClusterInstalledAppsModel(installAppVersionDTO, installAppVersionDTO.Environment.ClusterId)
		err = impl.clusterInstalledAppsRepository.Save(clusterInstalledAppsModel, tx)
		if err != nil {
			impl.logger.Errorw("error while creating cluster install app", "error", err)
			return nil, err
		}
		installAppVersions = append(installAppVersions, installAppVersionDTO)
	}
	if chartGroupInstallRequest.ChartGroupId > 0 {
		groupINstallationId, err := getInstallationId(installAppVersions)
		if err != nil {
			return nil, err
		}
		for _, installAppVersionDTO := range installAppVersions {
			chartGroupEntry := createChartGroupEntryObject(installAppVersionDTO, chartGroupInstallRequest.ChartGroupId, groupINstallationId)
			err := impl.chartGroupDeploymentRepository.Save(tx, chartGroupEntry)
			if err != nil {
				impl.logger.Errorw("DeployBulk, error in creating ChartGroupEntryObject", "err", err)
				return nil, err
			}
		}
	}
	//commit transaction
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("DeployBulk, error in tx commit", "err", err)
		return nil, err
	}
	//nats event

	for _, versions := range installAppVersions {
		_, err := impl.PerformDeployStage(versions.InstalledAppVersionId, versions.InstalledAppVersionHistoryId, chartGroupInstallRequest.UserId)
		if err != nil {
			impl.logger.Errorw("error in performing deploy stage", "deployPayload", versions, "err", err)
			_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(versions.InstalledAppVersionId, appStoreBean.QUE_ERROR)
			if err != nil {
				impl.logger.Errorw("error while bulk app-store deploy status update", "err", err)
			}
		}
	}
	// TODO refactoring: why empty obj ??
	return &ChartGroupInstallAppRes{}, nil
}

func (impl *ChartGroupServiceImpl) PerformDeployStage(installedAppVersionId int, installedAppVersionHistoryId int, userId int32) (*appStoreBean.InstallAppVersionDTO, error) {
	ctx := context.Background()
	installedAppVersion, err := impl.installAppService.GetInstalledAppVersion(installedAppVersionId, userId)
	if err != nil {
		return nil, err
	}
	installedAppVersion.InstalledAppVersionHistoryId = installedAppVersionHistoryId
	if util.IsAcdApp(installedAppVersion.DeploymentAppType) {
		//this method should only call in case of argo-integration installed and git-ops has configured

		timeline := &pipelineConfig.PipelineStatusTimeline{
			InstalledAppVersionHistoryId: installedAppVersion.InstalledAppVersionHistoryId,
			Status:                       timelineStatus.TIMELINE_STATUS_DEPLOYMENT_INITIATED,
			StatusDetail:                 "Deployment initiated successfully.",
			StatusTime:                   time.Now(),
			AuditLog: sql.AuditLog{
				CreatedBy: installedAppVersion.UserId,
				CreatedOn: time.Now(),
				UpdatedBy: installedAppVersion.UserId,
				UpdatedOn: time.Now(),
			},
		}
		err = impl.pipelineStatusTimelineService.SaveTimeline(timeline, nil)
		if err != nil {
			impl.logger.Errorw("error in creating timeline status for deployment initiation for this app store application", "err", err, "timeline", timeline)
		}
		_, err = impl.performDeployStageOnAcd(installedAppVersion, ctx, userId)
		if err != nil {
			impl.logger.Errorw("error", "err", err)
			return nil, err
		}
	} else if util.IsHelmApp(installedAppVersion.DeploymentAppType) {

		_, err = impl.appStoreDeploymentService.InstallAppByHelm(installedAppVersion, ctx)
		if err != nil {
			impl.logger.Errorw("error", "err", err)
			_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.HELM_ERROR)
			if err != nil {
				impl.logger.Errorw("error", "err", err)
				return nil, err
			}
			return nil, err
		}
	}

	//step 4 db operation status triggered
	_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.DEPLOY_SUCCESS)
	if err != nil {
		impl.logger.Errorw("error", "err", err)
		return nil, err
	}

	return installedAppVersion, nil
}

func (impl *ChartGroupServiceImpl) performDeployStageOnAcd(installedAppVersion *appStoreBean.InstallAppVersionDTO, ctx context.Context, userId int32) (*appStoreBean.InstallAppVersionDTO, error) {
	installedAppVersion.UpdateACDAppName()
	chartGitAttr := &commonBean.ChartGitAttribute{}
	if installedAppVersion.Status == appStoreBean.DEPLOY_INIT ||
		installedAppVersion.Status == appStoreBean.ENQUEUED ||
		installedAppVersion.Status == appStoreBean.QUE_ERROR ||
		installedAppVersion.Status == appStoreBean.GIT_ERROR {
		//step 2 git operation pull push
		//TODO: save git Timeline here
		appStoreAppVersion, err := impl.appStoreApplicationVersionRepository.FindById(installedAppVersion.AppStoreVersion)
		if err != nil {
			impl.logger.Errorw("fetching error", "err", err)
			return nil, err
		}
		err = impl.fullModeDeploymentService.CreateArgoRepoSecretIfNeeded(appStoreAppVersion)
		if err != nil {
			impl.logger.Errorw("error in creating argo app repository secret", "appStoreApplicationVersionId", appStoreAppVersion.Id, "err", err)
			return nil, err
		}
		appStoreGitOpsResponse, err := impl.fullModeDeploymentService.GenerateManifestAndPerformGitOperations(installedAppVersion, appStoreAppVersion)
		if err != nil {
			impl.logger.Errorw(" error", "err", err)
			_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.GIT_ERROR)
			if err != nil {
				impl.logger.Errorw(" error", "err", err)
				return nil, err
			}
			timeline := &pipelineConfig.PipelineStatusTimeline{
				InstalledAppVersionHistoryId: installedAppVersion.InstalledAppVersionHistoryId,
				Status:                       timelineStatus.TIMELINE_STATUS_GIT_COMMIT_FAILED,
				StatusDetail:                 fmt.Sprintf("Git commit failed - %v", err),
				StatusTime:                   time.Now(),
				AuditLog: sql.AuditLog{
					CreatedBy: installedAppVersion.UserId,
					CreatedOn: time.Now(),
					UpdatedBy: installedAppVersion.UserId,
					UpdatedOn: time.Now(),
				},
			}
			_ = impl.pipelineStatusTimelineService.SaveTimeline(timeline, nil)
			return nil, err
		}

		timeline := &pipelineConfig.PipelineStatusTimeline{
			InstalledAppVersionHistoryId: installedAppVersion.InstalledAppVersionHistoryId,
			Status:                       timelineStatus.TIMELINE_STATUS_GIT_COMMIT,
			StatusDetail:                 timelineStatus.TIMELINE_DESCRIPTION_ARGOCD_GIT_COMMIT,
			StatusTime:                   time.Now(),
		}
		timeline.CreateAuditLog(installedAppVersion.UserId)
		_ = impl.pipelineStatusTimelineService.SaveTimeline(timeline, nil)
		impl.logger.Infow("GIT SUCCESSFUL", "chartGitAttrDB", appStoreGitOpsResponse)
		_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.GIT_SUCCESS)
		if err != nil {
			impl.logger.Errorw(" error", "err", err)
			return nil, err
		}

		GitCommitSuccessTimeline := impl.pipelineStatusTimelineService.
			NewHelmAppDeploymentStatusTimelineDbObject(installedAppVersion.InstalledAppVersionHistoryId, timelineStatus.TIMELINE_STATUS_GIT_COMMIT, timelineStatus.TIMELINE_DESCRIPTION_ARGOCD_GIT_COMMIT, installedAppVersion.UserId)

		timelines := []*pipelineConfig.PipelineStatusTimeline{GitCommitSuccessTimeline}
		if impl.acdConfig.IsManualSyncEnabled() {
			ArgocdSyncInitiatedTimeline := impl.pipelineStatusTimelineService.
				NewHelmAppDeploymentStatusTimelineDbObject(installedAppVersion.InstalledAppVersionHistoryId, timelineStatus.TIMELINE_STATUS_ARGOCD_SYNC_INITIATED, timelineStatus.TIMELINE_DESCRIPTION_ARGOCD_SYNC_INITIATED, installedAppVersion.UserId)

			timelines = append(timelines, ArgocdSyncInitiatedTimeline)
		}

		dbConnection := impl.installedAppRepository.GetConnection()
		tx, err := dbConnection.Begin()
		if err != nil {
			impl.logger.Errorw("error in getting db connection for saving timelines", "err", err)
			return nil, err
		}
		err = impl.pipelineStatusTimelineService.SaveMultipleTimelinesIfNotAlreadyPresent(timelines, tx)
		if err != nil {
			impl.logger.Errorw("error in creating timeline status for deployment initiation for update of installedAppVersionHistoryId", "err", err, "installedAppVersionHistoryId", installedAppVersion.InstalledAppVersionHistoryId)
		}
		tx.Commit()
		// update build history for chart for argo_cd apps
		err = impl.appStoreDeploymentDBService.UpdateInstalledAppVersionHistoryWithGitHash(installedAppVersion.InstalledAppVersionHistoryId, installedAppVersion.GitHash, installedAppVersion.UserId)
		if err != nil {
			impl.logger.Errorw("error on updating history for chart deployment", "error", err, "installedAppVersion", installedAppVersion)
			return nil, err
		}
		installedAppVersion.GitHash = appStoreGitOpsResponse.GitHash
		chartGitAttr.RepoUrl = appStoreGitOpsResponse.ChartGitAttribute.RepoUrl
		chartGitAttr.TargetRevision = appStoreGitOpsResponse.ChartGitAttribute.TargetRevision
		chartGitAttr.ChartLocation = appStoreGitOpsResponse.ChartGitAttribute.ChartLocation
	} else {
		impl.logger.Infow("DB and GIT operation already done for this app and env, proceed for further step", "installedAppId", installedAppVersion.InstalledAppId, "existing status", installedAppVersion.Status)
		if installedAppVersion.Environment == nil {
			environment, err := impl.environmentService.GetExtendedEnvBeanById(installedAppVersion.EnvironmentId)
			if err != nil {
				impl.logger.Errorw("fetching environment error", "envId", installedAppVersion.EnvironmentId, "err", err)
				return nil, err
			}
			installedAppVersion.Environment = environment
		}
		adapter.UpdateAdditionalEnvDetails(installedAppVersion, installedAppVersion.Environment)

		chartGitAttr.RepoUrl = installedAppVersion.GitOpsRepoURL
		chartGitAttr.TargetRevision = installedAppVersion.GitOpsRepoURL
		chartGitAttr.ChartLocation = fmt.Sprintf("%s-%s", installedAppVersion.AppName, installedAppVersion.EnvironmentName)
	}

	if installedAppVersion.Status == appStoreBean.DEPLOY_INIT ||
		installedAppVersion.Status == appStoreBean.ENQUEUED ||
		installedAppVersion.Status == appStoreBean.QUE_ERROR ||
		installedAppVersion.Status == appStoreBean.GIT_ERROR ||
		installedAppVersion.Status == appStoreBean.GIT_SUCCESS ||
		installedAppVersion.Status == appStoreBean.ACD_ERROR {
		//step 3 acd operation register, sync
		_, err := impl.fullModeDeploymentService.InstallApp(installedAppVersion, chartGitAttr, ctx, nil)
		if err != nil {
			impl.logger.Errorw("error", "chartGitAttr", chartGitAttr, "err", err)
			_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.ACD_ERROR)
			if err != nil {
				impl.logger.Errorw("error", "err", err)
				return nil, err
			}
			return nil, err
		}
		impl.logger.Infow("ACD SUCCESSFUL", "chartGitAttr", chartGitAttr)
		_, err = impl.appStoreDeploymentDBService.AppStoreDeployOperationStatusUpdate(installedAppVersion.InstalledAppId, appStoreBean.ACD_SUCCESS)
		if err != nil {
			impl.logger.Errorw("error", "err", err)
			return nil, err
		}
	} else {
		impl.logger.Infow("DB and GIT and ACD operation already done for this app and env. process has been completed", "installedAppId", installedAppVersion.InstalledAppId, "existing status", installedAppVersion.Status)
	}
	return installedAppVersion, nil
}
