package helper

import (
	"bytes"
	"encoding/json"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	bean2 "github.com/devtron-labs/devtron/pkg/workflow/dag/bean"
	"time"
)

func GetBuildArtifact(request *bean2.CiArtifactWebhookRequest, ciPipelineId int, materialJson []byte, createdOn, updatedOn time.Time) *repository.CiArtifact {
	return &repository.CiArtifact{
		Image:              request.Image,
		ImageDigest:        request.ImageDigest,
		MaterialInfo:       string(materialJson),
		DataSource:         request.DataSource,
		PipelineId:         ciPipelineId,
		WorkflowId:         request.WorkflowId,
		ScanEnabled:        request.IsScanEnabled,
		IsArtifactUploaded: request.IsArtifactUploaded, // for backward compatibility
		Scanned:            false,
		AuditLog:           sql.AuditLog{CreatedBy: request.UserId, UpdatedBy: request.UserId, CreatedOn: createdOn, UpdatedOn: updatedOn},
	}
}

func GetMaterialInfoJson(materialInfo json.RawMessage) ([]byte, error) {
	var matJson []byte
	materialJson, err := materialInfo.MarshalJSON()
	if err != nil {
		return matJson, err
	}
	dst := new(bytes.Buffer)
	err = json.Compact(dst, materialJson)
	if err != nil {
		return matJson, err
	}
	matJson = dst.Bytes()
	return matJson, nil
}
