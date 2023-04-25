package types

import (
	"time"

	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

// Snapshot represents an EBS volume snapshot
type Snapshot struct {
	SnapshotID     string
	SourceVolumeID string
	Size           int64
	CreationTime   time.Time
	ReadyToUse     bool
}

type CreateSnapshotInput struct {
	_ struct{} `type:"structure"`

	ClientToken *string `type:"string"`

	Description *string `type:"string"`

	InstantAccess *bool `type:"boolean"`

	IsAuto *bool `type:"boolean"`

	ProjectName *string `type:"string"`

	RetentionDays *int32 `type:"int32"`

	// SnapshotName is a required field
	SnapshotName *string `type:"string" required:"true"`

	Tags []*TagForCreateSnapshotInput `type:"list"`

	// VolumeId is a required field
	VolumeId *string `type:"string" required:"true"`
}

type DeleteSnapshotInput struct {
	_ struct{} `type:"structure"`

	ClientToken *string `type:"string"`

	// SnapshotId is a required field
	SnapshotId *string `type:"string" required:"true"`
}

type DescribeSnapshotsInput struct {
	_ struct{} `type:"structure"`

	Encrypted *bool `type:"boolean"`

	Filter []*FilterForDescribeSnapshotsInput `type:"list"`

	KMSKeyId *string `type:"string"`

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `max:"100" type:"int32"`

	ProjectName *string `type:"string"`

	SnapshotIds []*string `type:"list"`

	SnapshotName *string `type:"string"`

	SnapshotStatus []*string `type:"list"`

	SysTagVisible *bool `type:"boolean"`

	TagFilters []*TagFilterForDescribeSnapshotsInput `type:"list"`

	VolumeId *string `type:"string"`

	ZoneId *string `type:"string"`
}

type TagForCreateSnapshotInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}

type FilterForDescribeSnapshotsInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}

type TagFilterForDescribeSnapshotsInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Values []*string `type:"list"`
}

type CreateSnapshotOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	SnapshotId *string `type:"string"`
}

type DeleteSnapshotOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata
}

type DescribeSnapshotsOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `type:"int32"`

	RequestId *string `type:"string"`

	Snapshots []*SnapshotForDescribeSnapshotsOutput `type:"list"`

	TotalCount *int32 `type:"int32"`
}

type SnapshotForDescribeSnapshotsOutput struct {
	_ struct{} `type:"structure"`

	CreationTime *string `type:"string"`

	Description *string `type:"string"`

	Encrypted *bool `type:"boolean"`

	KMSKeyId *string `type:"string"`

	ProjectName *string `type:"string"`

	RetentionDays *int32 `type:"int32"`

	SnapshotId *string `type:"string"`

	SnapshotName *string `type:"string"`

	SnapshotType *string `type:"string"`

	Status *string `type:"string"`

	Tags []*TagForDescribeSnapshotsOutput `type:"list"`

	VolumeId *string `type:"string"`

	VolumeKind *string `type:"string"`

	VolumeName *string `type:"string"`

	VolumeSize *int64 `type:"int64"`

	VolumeStatus *string `type:"string"`

	VolumeType *string `type:"string"`

	ZoneId *string `type:"string"`
}

type TagForDescribeSnapshotsOutput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}
