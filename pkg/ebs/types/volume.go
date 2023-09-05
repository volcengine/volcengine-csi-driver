package types

import (
	"encoding/json"

	"github.com/volcengine/volcengine-go-sdk/volcengine/response"
)

const (
	GB int64 = 1 << 30
)

const (
	EBSStatusAvailable = "available"
	EBSStatusAttached  = "attached"
	EBSStatusDeleted   = "deleted"
	EBSStatusDetaching = "detaching"
	EBSStatusExtending = "extending"
	EBSStatusUnknown   = "unknown"
)

type Status uint32

const (
	StatusAvailable Status = iota
	StatusAttached
	StatusDeleted
	StatusDetaching
	StatusExtending
	StatusUnknown
)

type Volume struct {
	Id         string
	Name       string
	Status     Status
	Capacity   int64
	NodeId     string
	ZoneId     string
	VolumeType string
}

func (v *Volume) Attached() bool {
	if v == nil {
		return false
	}
	return v.Status == StatusAttached
}

func (v *Volume) Available() bool {
	if v == nil {
		return false
	}
	return v.Status == StatusAvailable
}

func (v *Volume) Deleted() bool {
	if v == nil {
		return false
	}
	return v.Status == StatusDeleted
}

func (v *Volume) Detaching() bool {
	if v == nil {
		return false
	}
	return v.Status == StatusDetaching
}

func StatusFromString(status string) Status {
	switch status {
	case EBSStatusAvailable:
		return StatusAvailable
	case EBSStatusAttached:
		return StatusAttached
	case EBSStatusDeleted:
		return StatusDeleted
	case EBSStatusDetaching:
		return StatusDetaching
	case EBSStatusExtending:
		return StatusExtending
	}
	return StatusUnknown
}

func StatusToString(status Status) string {
	switch status {
	case StatusAvailable:
		return EBSStatusAvailable
	case StatusAttached:
		return EBSStatusAttached
	case StatusDeleted:
		return EBSStatusDeleted
	}
	return EBSStatusUnknown
}

type CreateVolumeInput struct {
	_ struct{} `type:"structure"`

	AccountId *string `type:"string"`

	AutoRenew *bool `type:"boolean"`

	ClientToken *string `type:"string"`

	Description *string `type:"string"`

	InstanceId *string `type:"string"`

	Kind *string `type:"string"`

	Period *string `type:"string"`

	ProjectName *string `type:"string"`

	RenewCycle *int32 `type:"int32"`

	RenewTimes *int32 `type:"int32"`

	Size *json.Number `type:"json_number"`

	SnapshotId *string `type:"string"`

	StoragePoolId *string `type:"string"`

	Tags []*TagForCreateVolumeInput `type:"list"`

	Times *int32 `type:"int32"`

	VolumeChargeType *string `type:"string"`

	VolumeName *string `type:"string"`

	VolumeType *string `type:"string"`

	ZoneId *string `type:"string"`
}

type TagForCreateVolumeInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}

type CreateVolumeOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	VolumeId *string `type:"string"`
}

type DescribeVolumesInput struct {
	_ struct{} `type:"structure"`

	AccountId *string `type:"string"`

	AutoSnapshotPolicyId *string `type:"string"`

	Encrypted *bool `type:"boolean"`

	FieldMask *FieldMaskForDescribeVolumesInput `type:"structure"`

	InstanceId *string `type:"string"`

	KMSKeyId *string `type:"string"`

	Kind *string `type:"string"`

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `type:"int32"`

	ProjectName *string `type:"string"`

	SysTagVisible *bool `type:"boolean"`

	TagFilters []*TagFilterForDescribeVolumesInput `type:"list"`

	VolumeIds []*string `type:"list"`

	VolumeName *string `type:"string"`

	VolumeStatus *string `type:"string"`

	VolumeType *string `type:"string"`

	ZoneId *string `type:"string"`
}

type FieldMaskForDescribeVolumesInput struct {
	_ struct{} `type:"structure"`

	Paths *string `type:"string"`
}

type TagFilterForDescribeVolumesInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Values []*string `type:"list"`
}

type DescribeVolumesOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `type:"int32"`

	TotalCount *int32 `type:"int32"`

	Volumes []*VolumeForDescribeVolumesOutput `type:"list"`
}

type VolumeForDescribeVolumesOutput struct {
	_ struct{} `type:"structure"`

	AutoSnapshotPolicyId *string `type:"string"`

	AutoSnapshotPolicyName *string `type:"string"`

	BillingType *int32 `type:"int32"`

	CreatedAt *string `type:"string"`

	DeleteWithInstance *bool `type:"boolean"`

	Description *string `type:"string"`

	DeviceName *string `type:"string"`

	Encrypted *bool `type:"boolean"`

	ErrorDetail *string `type:"string"`

	ExpiredTime *string `type:"string"`

	ImageId *string `type:"string"`

	InstanceId *string `type:"string"`

	KMSKeyId *string `type:"string"`

	Kind *string `type:"string"`

	OverdueReclaimTime *string `type:"string"`

	OverdueTime *string `type:"string"`

	PayType *string `type:"string"`

	ProjectName *string `type:"string"`

	RenewType *int32 `type:"int32"`

	Size *json.Number `type:"json_number"`

	SnapshotCount *int32 `type:"int32"`

	SourceSnapshotId *string `type:"string"`

	Status *string `type:"string"`

	Tags []*TagForDescribeVolumesOutput `type:"list"`

	TradeStatus *int32 `type:"int32"`

	UpdatedAt *string `type:"string"`

	VolumeId *string `type:"string"`

	VolumeName *string `type:"string"`

	VolumeType *string `type:"string"`

	ZoneId *string `type:"string"`
}

type TagForDescribeVolumesOutput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}

type DescribeInstanceTypesInput struct {
	_ struct{} `type:"structure"`

	InstanceTypeIds []*string `type:"list"`

	InstanceTypes []*string `type:"list"`

	MaxResults *int32 `type:"int32"`

	NextToken *string `type:"string"`
}

type DescribeInstanceTypesOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	InstanceTypes []*InstanceTypeForDescribeInstanceTypesOutput `type:"list"`

	NextToken *string `type:"string"`

	TotalCount *int32 `type:"int32"`
}

type InstanceTypeForDescribeInstanceTypesOutput struct {
	_ struct{} `type:"structure"`

	InstanceTypeFamily *string `type:"string"`

	InstanceTypeId *string `type:"string"`

	LocalVolumes []*LocalVolumeForDescribeInstanceTypesOutput `type:"list"`

	Volume *VolumeForDescribeInstanceTypesOutput `type:"structure"`
}

type LocalVolumeForDescribeInstanceTypesOutput struct {
	_ struct{} `type:"structure"`

	Count *int32 `type:"int32"`

	Size *int32 `type:"int32"`

	VolumeType *string `type:"string"`
}

type VolumeForDescribeInstanceTypesOutput struct {
	_ struct{} `type:"structure"`

	MaximumCount *int32 `type:"int32"`

	SupportedVolumeTypes []*string `type:"list"`
}
