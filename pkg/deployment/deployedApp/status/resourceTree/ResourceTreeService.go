package resourceTree

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/gitops-engine/pkg/health"
	k8sCommonBean "github.com/devtron-labs/common-lib/utils/k8s/commonBean"
	k8sObjectUtils "github.com/devtron-labs/common-lib/utils/k8sObjectsUtil"
	"github.com/devtron-labs/devtron/api/bean/AppView"
	"github.com/devtron-labs/devtron/api/helm-app/gRPC"
	"github.com/devtron-labs/devtron/api/helm-app/service/read"
	argoApplication "github.com/devtron-labs/devtron/client/argocdServer/bean"
	"github.com/devtron-labs/devtron/client/cron"
	"github.com/devtron-labs/devtron/internal/constants"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/app"
	"github.com/devtron-labs/devtron/pkg/appStatus"
	argoApplication2 "github.com/devtron-labs/devtron/pkg/argoApplication"
	commonBean "github.com/devtron-labs/devtron/pkg/deployment/common/bean"
	"github.com/devtron-labs/devtron/pkg/k8s"
	application2 "github.com/devtron-labs/devtron/pkg/k8s/application"
	util2 "github.com/devtron-labs/devtron/util"
	"go.uber.org/zap"
	"strconv"
	"time"
)

type Service interface {
	FetchResourceTree(ctx context.Context, appId int, envId int, cdPipeline *pipelineConfig.Pipeline,
		deploymentConfig *commonBean.DeploymentConfig) (map[string]interface{}, error)

	//ent
	FetchResourceTreeWithDrift(ctx context.Context, appId int, envId int, cdPipeline *pipelineConfig.Pipeline,
		deploymentConfig *commonBean.DeploymentConfig) (map[string]interface{}, error)
}

type ServiceImpl struct {
	logger                           *zap.SugaredLogger
	appListingService                app.AppListingService
	appStatusService                 appStatus.AppStatusService
	argoApplicationService           argoApplication2.ArgoApplicationService
	cdApplicationStatusUpdateHandler cron.CdApplicationStatusUpdateHandler
	helmAppClient                    gRPC.HelmAppClient
	helmAppReadService               read.HelmAppReadService
	k8sApplicationService            application2.K8sApplicationService
	k8sCommonService                 k8s.K8sCommonService
}

func NewServiceImpl(logger *zap.SugaredLogger,
	appListingService app.AppListingService,
	appStatusService appStatus.AppStatusService,
	argoApplicationService argoApplication2.ArgoApplicationService,
	cdApplicationStatusUpdateHandler cron.CdApplicationStatusUpdateHandler,
	helmAppClient gRPC.HelmAppClient,
	helmAppReadService read.HelmAppReadService,
	k8sApplicationService application2.K8sApplicationService,
	k8sCommonService k8s.K8sCommonService) *ServiceImpl {
	serviceImpl := &ServiceImpl{
		logger:                           logger,
		appListingService:                appListingService,
		appStatusService:                 appStatusService,
		argoApplicationService:           argoApplicationService,
		cdApplicationStatusUpdateHandler: cdApplicationStatusUpdateHandler,
		helmAppClient:                    helmAppClient,
		helmAppReadService:               helmAppReadService,
		k8sApplicationService:            k8sApplicationService,
		k8sCommonService:                 k8sCommonService,
	}
	return serviceImpl
}

func (impl ServiceImpl) FetchResourceTree(ctx context.Context, appId int, envId int, cdPipeline *pipelineConfig.Pipeline,
	deploymentConfig *commonBean.DeploymentConfig) (map[string]interface{}, error) {
	var resourceTree map[string]interface{}
	if !cdPipeline.DeploymentAppCreated {
		impl.logger.Infow("deployment for this pipeline does not exist", "pipelineId", cdPipeline.Id)
		return resourceTree, nil
	}

	if len(cdPipeline.DeploymentAppName) > 0 && cdPipeline.EnvironmentId > 0 && util.IsAcdApp(deploymentConfig.DeploymentAppType) {
		// RBAC enforcer Ends
		query := &application.ResourcesQuery{
			ApplicationName: &cdPipeline.DeploymentAppName,
		}
		start := time.Now()
		resp, err := impl.argoApplicationService.ResourceTree(ctx, query)
		elapsed := time.Since(start)
		impl.logger.Debugw("FetchAppDetailsV2, time elapsed in fetching application for environment ", "elapsed", elapsed, "appId", appId, "envId", envId)
		if err != nil {
			impl.logger.Errorw("service err, FetchAppDetailsV2, resource tree", "err", err, "app", appId, "env", envId)
			internalMsg := fmt.Sprintf("%s, err:- %s", constants.UnableToFetchResourceTreeForAcdErrMsg, err.Error())
			clientCode, _ := util.GetClientDetailedError(err)
			httpStatusCode := clientCode.GetHttpStatusCodeForGivenGrpcCode()
			err = &util.ApiError{
				HttpStatusCode:  httpStatusCode,
				Code:            constants.AppDetailResourceTreeNotFound,
				InternalMessage: internalMsg,
				UserMessage:     "Error fetching detail, if you have recently created this deployment pipeline please try after sometime.",
			}
			return resourceTree, err
		}

		// we currently add appId and envId as labels for devtron apps deployed via acd
		label := fmt.Sprintf("appId=%v,envId=%v", cdPipeline.AppId, cdPipeline.EnvironmentId)
		pods, err := impl.k8sApplicationService.GetPodListByLabel(cdPipeline.Environment.ClusterId, cdPipeline.Environment.Namespace, label)
		if err != nil {
			impl.logger.Errorw("error in getting pods by label", "err", err, "clusterId", cdPipeline.Environment.ClusterId, "namespace", cdPipeline.Environment.Namespace, "label", label)
			return resourceTree, err
		}
		ephemeralContainersMap := k8sObjectUtils.ExtractEphemeralContainers(pods)
		for _, metaData := range resp.PodMetadata {
			metaData.EphemeralContainers = ephemeralContainersMap[metaData.Name]
		}

		if resp.Status == string(health.HealthStatusHealthy) {
			status, err := impl.appListingService.ISLastReleaseStopType(appId, envId)
			if err != nil {
				impl.logger.Errorw("service err, FetchAppDetailsV2", "err", err, "app", appId, "env", envId)
			} else if status {
				resp.Status = argoApplication.HIBERNATING
			}
		}
		if resp.Status == string(health.HealthStatusDegraded) {
			count, err := impl.appListingService.GetReleaseCount(appId, envId)
			if err != nil {
				impl.logger.Errorw("service err, FetchAppDetailsV2, release count", "err", err, "app", appId, "env", envId)
			} else if count == 0 {
				resp.Status = app.NotDeployed
			}
		}
		resourceTree = util2.InterfaceToMapAdapter(resp)
		go func() {
			if resp.Status == string(health.HealthStatusHealthy) {
				err = impl.cdApplicationStatusUpdateHandler.SyncPipelineStatusForResourceTreeCall(cdPipeline)
				if err != nil {
					impl.logger.Errorw("error in syncing pipeline status", "err", err)
				}
			}
			// updating app_status table here
			err = impl.appStatusService.UpdateStatusWithAppIdEnvId(appId, envId, resp.Status)
			if err != nil {
				impl.logger.Warnw("error in updating app status", "err", err, "appId", cdPipeline.AppId, "envId", cdPipeline.EnvironmentId)
			}
		}()

	} else if len(cdPipeline.DeploymentAppName) > 0 && cdPipeline.EnvironmentId > 0 && util.IsHelmApp(deploymentConfig.DeploymentAppType) {
		config, err := impl.helmAppReadService.GetClusterConf(cdPipeline.Environment.ClusterId)
		if err != nil {
			impl.logger.Errorw("error in fetching cluster detail", "err", err)
		}
		req := &gRPC.AppDetailRequest{
			ClusterConfig: config,
			Namespace:     cdPipeline.Environment.Namespace,
			ReleaseName:   cdPipeline.DeploymentAppName,
		}
		detail, err := impl.helmAppClient.GetAppDetail(context.Background(), req)
		if err != nil {
			impl.logger.Errorw("error in fetching app detail", "err", err)
		}
		if detail != nil && detail.ReleaseExist {
			resourceTree = util2.InterfaceToMapAdapter(detail.ResourceTreeResponse)
			releaseStatus := util2.InterfaceToMapAdapter(detail.ReleaseStatus)
			applicationStatus := detail.ApplicationStatus
			resourceTree["releaseStatus"] = releaseStatus
			resourceTree["status"] = applicationStatus
			if applicationStatus == argoApplication.Healthy {
				status, err := impl.appListingService.ISLastReleaseStopType(appId, envId)
				if err != nil {
					impl.logger.Errorw("service err, FetchAppDetailsV2", "err", err, "app", appId, "env", envId)
				} else if status {
					resourceTree["status"] = argoApplication.HIBERNATING
				}
			}
			impl.logger.Warnw("appName and envName not found - avoiding resource tree call", "app", cdPipeline.DeploymentAppName, "env", cdPipeline.Environment.Name)
		}
	} else {
		impl.logger.Warnw("appName and envName not found - avoiding resource tree call", "app", cdPipeline.DeploymentAppName, "env", cdPipeline.Environment.Name)
	}
	if resourceTree != nil {
		version, err := impl.k8sCommonService.GetK8sServerVersion(cdPipeline.Environment.ClusterId)
		if err != nil {
			impl.logger.Errorw("error in fetching k8s version in resource tree call fetching", "clusterId", cdPipeline.Environment.ClusterId, "err", err)
		} else {
			resourceTree["serverVersion"] = version.String()
		}
	}
	k8sAppDetail := AppView.AppDetailContainer{
		DeploymentDetailContainer: AppView.DeploymentDetailContainer{
			ClusterId: cdPipeline.Environment.ClusterId,
			Namespace: cdPipeline.Environment.Namespace,
		},
	}
	clusterIdString := strconv.Itoa(cdPipeline.Environment.ClusterId)
	validRequest := impl.k8sCommonService.FilterK8sResources(ctx, resourceTree, k8sAppDetail, clusterIdString, []string{k8sCommonBean.ServiceKind, k8sCommonBean.EndpointsKind, k8sCommonBean.IngressKind}, "")
	resp, err := impl.k8sCommonService.GetManifestsByBatch(ctx, validRequest)
	if err != nil {
		impl.logger.Errorw("error in getting manifest by batch", "err", err, "clusterId", clusterIdString)
		httpStatus, ok := util.IsErrorContextCancelledOrDeadlineExceeded(err)
		if ok {
			return nil, &util.ApiError{HttpStatusCode: httpStatus, Code: strconv.Itoa(httpStatus), InternalMessage: err.Error()}
		}
		return nil, err
	}
	newResourceTree := impl.k8sCommonService.PortNumberExtraction(resp, resourceTree)
	return newResourceTree, nil
}
