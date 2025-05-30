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

package repository

import (
	"github.com/devtron-labs/devtron/internal/sql/constants"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
)

type GitProvider struct {
	tableName             struct{}           `sql:"git_provider" pg:",discard_unknown_columns"`
	Id                    int                `sql:"id,pk"`
	Name                  string             `sql:"name,notnull"`
	Url                   string             `sql:"url,notnull"`
	UserName              string             `sql:"user_name"`
	Password              string             `sql:"password"`
	SshPrivateKey         string             `sql:"ssh_private_key"`
	AccessToken           string             `sql:"access_token"`
	AuthMode              constants.AuthMode `sql:"auth_mode,notnull"`
	Active                bool               `sql:"active,notnull"`
	Deleted               bool               `sql:"deleted,notnull"`
	GitHostId             int                `sql:"git_host_id"` //id stored in db git_host( foreign key)
	TlsCert               string             `sql:"tls_cert"`
	TlsKey                string             `sql:"tls_key"`
	CaCert                string             `sql:"ca_cert"`
	EnableTLSVerification bool               `sql:"enable_tls_verification"`
	sql.AuditLog
}

type GitProviderRepository interface {
	Save(gitProvider *GitProvider) error
	ProviderExists(url string) (bool, error)
	FindAllActiveForAutocomplete() ([]GitProvider, error)
	FindAll() ([]GitProvider, error)
	FindAllGitProviderCount() (int, error)
	FindOne(providerId string) (GitProvider, error)
	FindByUrl(providerUrl string) (GitProvider, error)
	Update(gitProvider *GitProvider) error
	MarkProviderDeleted(gitProvider *GitProvider) error
}

type GitProviderRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewGitProviderRepositoryImpl(dbConnection *pg.DB) *GitProviderRepositoryImpl {
	return &GitProviderRepositoryImpl{dbConnection: dbConnection}
}

func (impl GitProviderRepositoryImpl) Save(gitProvider *GitProvider) error {
	err := impl.dbConnection.Insert(gitProvider)
	return err
}

func (impl GitProviderRepositoryImpl) ProviderExists(url string) (bool, error) {
	provider := &GitProvider{}
	exists, err := impl.dbConnection.
		Model(provider).
		Where("url = ?", url).
		Where("deleted = ?", false).
		Exists()
	return exists, err
}

func (impl GitProviderRepositoryImpl) FindAllActiveForAutocomplete() ([]GitProvider, error) {
	var providers []GitProvider
	err := impl.dbConnection.Model(&providers).
		Where("active = ?", true).Column("id", "name", "url", "auth_mode").
		Where("deleted = ?", false).Select()
	return providers, err
}

func (impl GitProviderRepositoryImpl) FindAll() ([]GitProvider, error) {
	var providers []GitProvider
	err := impl.dbConnection.Model(&providers).
		Where("deleted = ?", false).Select()
	return providers, err
}
func (impl GitProviderRepositoryImpl) FindAllGitProviderCount() (int, error) {
	gitProviderCount, err := impl.dbConnection.Model(&GitProvider{}).
		Where("deleted = ?", false).Count()
	return gitProviderCount, err
}

func (impl GitProviderRepositoryImpl) FindOne(providerId string) (GitProvider, error) {
	var provider GitProvider
	err := impl.dbConnection.Model(&provider).
		Where("id = ?", providerId).
		Where("deleted = ?", false).
		Select()
	return provider, err
}

func (impl GitProviderRepositoryImpl) FindByUrl(providerUrl string) (GitProvider, error) {
	var provider GitProvider
	err := impl.dbConnection.Model(&provider).
		Where("url = ?", providerUrl).Where("active = ?", true).
		Where("deleted = ?", false).Select()
	return provider, err
}

func (impl GitProviderRepositoryImpl) Update(gitProvider *GitProvider) error {
	err := impl.dbConnection.Update(gitProvider)
	return err
}

func (impl GitProviderRepositoryImpl) MarkProviderDeleted(gitProvider *GitProvider) error {
	gitProvider.Deleted = true
	return impl.dbConnection.Update(gitProvider)
}
