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

package user

import (
	"encoding/json"
	"fmt"
	bean2 "github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin/bean"
	bean3 "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"net/http"
	"strings"

	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/util"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

type UserAuthHandler interface {
	LoginHandler(w http.ResponseWriter, r *http.Request)
	CallbackHandler(w http.ResponseWriter, r *http.Request)
	RefreshTokenHandler(w http.ResponseWriter, r *http.Request)
	AddDefaultPolicyAndRoles(w http.ResponseWriter, r *http.Request)
	AuthVerification(w http.ResponseWriter, r *http.Request)
	AuthVerificationV2(w http.ResponseWriter, r *http.Request)
}

type UserAuthHandlerImpl struct {
	userAuthService user.UserAuthService
	validator       *validator.Validate
	logger          *zap.SugaredLogger
	enforcer        casbin.Enforcer
}

func NewUserAuthHandlerImpl(
	userAuthService user.UserAuthService,
	validator *validator.Validate,
	logger *zap.SugaredLogger, enforcer casbin.Enforcer) *UserAuthHandlerImpl {
	userAuthHandler := &UserAuthHandlerImpl{
		userAuthService: userAuthService,
		validator:       validator,
		logger:          logger,
		enforcer:        enforcer,
	}
	return userAuthHandler
}

func (handler UserAuthHandlerImpl) LoginHandler(w http.ResponseWriter, r *http.Request) {
	up := &userNamePassword{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(up)
	if err != nil {
		handler.logger.Errorw("request err, LoginHandler", "err", err, "payload", up)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
	}

	err = handler.validator.Struct(up)
	if err != nil {
		handler.logger.Errorw("validation err, LoginHandler", "err", err, "payload", up)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	//token, err := handler.loginService.CreateLoginSession(up.Username, up.Password)
	clientIp := util.GetClientIP(r)
	token, err := handler.userAuthService.HandleLoginWithClientIp(r.Context(), up.Username, up.Password, clientIp)
	if err != nil {
		common.WriteJsonResp(w, fmt.Errorf("invalid username or password"), nil, http.StatusForbidden)
		return
	}
	response := make(map[string]interface{})
	response["token"] = token
	http.SetCookie(w, &http.Cookie{Name: "argocd.token", Value: token, Path: "/"})
	common.WriteJsonResp(w, nil, response, http.StatusOK)
}

func (handler UserAuthHandlerImpl) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	handler.userAuthService.HandleDexCallback(w, r)
}

func (handler UserAuthHandlerImpl) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	handler.userAuthService.HandleRefresh(w, r)
}

func (handler UserAuthHandlerImpl) AddDefaultPolicyAndRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	team := vars["team"]
	app := vars["app"]
	env := vars["env"]
	handler.logger.Infow("request payload, AddDefaultPolicyAndRoles", "team", team, "app", app, "env", env)
	adminPolicies := "{\n    \"data\": [\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:admin_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"applications\",\n            \"act\": \"*\",\n            \"obj\": \"<TEAM_OBJ>/<APP_OBJ>\"\n        },\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:admin_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"environment\",\n            \"act\": \"trigger\",\n            \"obj\": \"<ENV_OBJ>/<APP_OBJ>\"\n        },\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:admin_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"team\",\n            \"act\": \"get\",\n            \"obj\": \"<TEAM_OBJ>\"\n        }\n    ]\n}"
	triggerPolicies := "{\n    \"data\": [\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:trigger_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"applications\",\n            \"act\": \"get\",\n            \"obj\": \"<TEAM_OBJ>/<APP_OBJ>\"\n        },\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:trigger_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"applications\",\n            \"act\": \"trigger\",\n            \"obj\": \"<TEAM_OBJ>/<APP_OBJ>\"\n        },\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:trigger_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"environment\",\n            \"act\": \"trigger\",\n            \"obj\": \"<ENV_OBJ>/<APP_OBJ>\"\n        }\n    ]\n}"
	viewPolicies := "{\n    \"data\": [\n        {\n            \"type\": \"p\",\n            \"sub\": \"role:view_<TEAM>_<ENV>_<APP>\",\n            \"res\": \"applications\",\n            \"act\": \"get\",\n            \"obj\": \"<TEAM_OBJ>/<APP_OBJ>\"\n        }\n    ]\n}"

	adminPolicies = strings.ReplaceAll(adminPolicies, "<TEAM>", team)
	adminPolicies = strings.ReplaceAll(adminPolicies, "<ENV>", env)
	adminPolicies = strings.ReplaceAll(adminPolicies, "<APP>", app)

	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<TEAM>", team)
	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<ENV>", env)
	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<APP>", app)

	viewPolicies = strings.ReplaceAll(viewPolicies, "<TEAM>", team)
	viewPolicies = strings.ReplaceAll(viewPolicies, "<ENV>", env)
	viewPolicies = strings.ReplaceAll(viewPolicies, "<APP>", app)

	//for START in Casbin Object
	teamObj := team
	envObj := env
	appObj := app
	if len(teamObj) == 0 {
		teamObj = "*"
	}
	if len(envObj) == 0 {
		envObj = "*"
	}
	if len(appObj) == 0 {
		appObj = "*"
	}
	adminPolicies = strings.ReplaceAll(adminPolicies, "<TEAM_OBJ>", teamObj)
	adminPolicies = strings.ReplaceAll(adminPolicies, "<ENV_OBJ>", envObj)
	adminPolicies = strings.ReplaceAll(adminPolicies, "<APP_OBJ>", appObj)

	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<TEAM_OBJ>", teamObj)
	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<ENV_OBJ>", envObj)
	triggerPolicies = strings.ReplaceAll(triggerPolicies, "<APP_OBJ>", appObj)

	viewPolicies = strings.ReplaceAll(viewPolicies, "<TEAM_OBJ>", teamObj)
	viewPolicies = strings.ReplaceAll(viewPolicies, "<ENV_OBJ>", envObj)
	viewPolicies = strings.ReplaceAll(viewPolicies, "<APP_OBJ>", appObj)
	//for START in Casbin Object Ends Here
	//loading policy for safety
	casbin.LoadPolicy()
	var policies []bean2.Policy
	var policiesAdmin bean.PolicyRequest
	err := json.Unmarshal([]byte(adminPolicies), &policiesAdmin)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", policiesAdmin)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Debugw("request payload, AddDefaultPolicyAndRoles", "policiesAdmin", policiesAdmin)
	policies = append(policies, policiesAdmin.Data...)
	var policiesTrigger bean.PolicyRequest
	err = json.Unmarshal([]byte(triggerPolicies), &policiesTrigger)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", policiesTrigger)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Debugw("request payload, AddDefaultPolicyAndRoles", "policiesTrigger", policiesTrigger)
	policies = append(policies, policiesTrigger.Data...)
	var policiesView bean.PolicyRequest
	err = json.Unmarshal([]byte(viewPolicies), &policiesView)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", policiesView)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.logger.Debugw("request payload, AddDefaultPolicyAndRoles", "policiesView", policiesView)
	policies = append(policies, policiesView.Data...)
	casbin.AddPolicy(policies)
	//loading policy for syncing orchestrator to casbin with newly added policies
	casbin.LoadPolicy()
	//Creating ROLES
	roleAdmin := "{\n    \"role\": \"role:admin_<TEAM>_<ENV>_<APP>\",\n    \"casbinSubjects\": [\n        \"role:admin_<TEAM>_<ENV>_<APP>\"\n    ],\n    \"team\": \"<TEAM>\",\n    \"application\": \"<APP>\",\n    \"environment\": \"<ENV>\",\n    \"action\": \"*\"\n}"
	roleTrigger := "{\n    \"role\": \"role:trigger_<TEAM>_<ENV>_<APP>\",\n    \"casbinSubjects\": [\n        \"role:trigger_<TEAM>_<ENV>_<APP>\"\n    ],\n    \"team\": \"<TEAM>\",\n    \"application\": \"<APP>\",\n    \"environment\": \"<ENV>\",\n    \"action\": \"trigger\"\n}"
	roleView := "{\n    \"role\": \"role:view_<TEAM>_<ENV>_<APP>\",\n    \"casbinSubjects\": [\n        \"role:view_<TEAM>_<ENV>_<APP>\"\n    ],\n    \"team\": \"<TEAM>\",\n    \"application\": \"<APP>\",\n    \"environment\": \"<ENV>\",\n    \"action\": \"view\"\n}"
	roleAdmin = strings.ReplaceAll(roleAdmin, "<TEAM>", team)
	roleAdmin = strings.ReplaceAll(roleAdmin, "<ENV>", env)
	roleAdmin = strings.ReplaceAll(roleAdmin, "<APP>", app)

	roleTrigger = strings.ReplaceAll(roleTrigger, "<TEAM>", team)
	roleTrigger = strings.ReplaceAll(roleTrigger, "<ENV>", env)
	roleTrigger = strings.ReplaceAll(roleTrigger, "<APP>", app)

	roleView = strings.ReplaceAll(roleView, "<TEAM>", team)
	roleView = strings.ReplaceAll(roleView, "<ENV>", env)
	roleView = strings.ReplaceAll(roleView, "<APP>", app)

	var roleAdminData bean3.RoleData
	err = json.Unmarshal([]byte(roleAdmin), &roleAdminData)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", roleAdminData)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	_, err = handler.userAuthService.CreateRole(&roleAdminData)
	if err != nil {
		handler.logger.Errorw("service err, AddDefaultPolicyAndRoles", "err", err, "payload", roleAdminData)
		common.WriteJsonResp(w, err, "Role Creation Failed", http.StatusInternalServerError)
		return
	}

	var roleTriggerData bean3.RoleData
	err = json.Unmarshal([]byte(roleTrigger), &roleTriggerData)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", roleTriggerData)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	_, err = handler.userAuthService.CreateRole(&roleTriggerData)
	if err != nil {
		handler.logger.Errorw("service err, AddDefaultPolicyAndRoles", "err", err, "payload", roleTriggerData)
		common.WriteJsonResp(w, err, "Role Creation Failed", http.StatusInternalServerError)
		return
	}

	var roleViewData bean3.RoleData
	err = json.Unmarshal([]byte(roleView), &roleViewData)
	if err != nil {
		handler.logger.Errorw("request err, AddDefaultPolicyAndRoles", "err", err, "payload", roleViewData)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	_, err = handler.userAuthService.CreateRole(&roleViewData)
	if err != nil {
		handler.logger.Errorw("service err, AddDefaultPolicyAndRoles", "err", err, "payload", roleTriggerData)
		common.WriteJsonResp(w, err, "Role Creation Failed", http.StatusInternalServerError)
		return
	}

}
func (handler UserAuthHandlerImpl) AuthVerification(w http.ResponseWriter, r *http.Request) {
	verified, _, err := handler.userAuthService.AuthVerification(r)
	if err != nil {
		handler.logger.Errorw("service err, AuthVerification", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, verified, http.StatusOK)
}

func (handler UserAuthHandlerImpl) AuthVerificationV2(w http.ResponseWriter, r *http.Request) {
	isSuperAdmin := false
	token := r.Header.Get("token")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); ok {
		isSuperAdmin = true
	}
	response := make(map[string]interface{})
	verified, emailId, err := handler.userAuthService.AuthVerification(r)
	if err != nil {
		handler.logger.Errorw("service err, AuthVerification", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	response["isSuperAdmin"] = isSuperAdmin
	response["isVerified"] = verified
	response["emailId"] = emailId
	common.WriteJsonResp(w, nil, response, http.StatusOK)
}
