package types

import "github.com/volcengine/volcengine-go-sdk/volcengine/response"

type DescribeInstancesInput struct {
	_ struct{} `type:"structure"`

	AccountId *string `type:"string"`

	DeploymentSetGroupNumbers []*int64 `type:"list"`

	DeploymentSetIds []*string `type:"list"`

	DescribeInstanceReqOption *DescribeInstanceReqOptionForDescribeInstancesInput `type:"structure"`

	FieldMask *FieldMaskForDescribeInstancesInput `type:"structure"`

	HpcClusterId *string `type:"string"`

	InstanceChargeType *string `type:"string"`

	InstanceIds []*string `type:"list"`

	InstanceName *string `type:"string"`

	InstanceType *string `type:"string"`

	InstanceTypeFamilies []*string `type:"list"`

	InstanceTypeId *string `type:"string"`

	InstanceTypeIds []*string `type:"list"`

	InstanceTypes []*string `type:"list"`

	KeyPairName *string `type:"string"`

	MaxResults *int32 `type:"int32"`

	NetworkInterfaceType *string `type:"string"`

	NextToken *string `type:"string"`

	NotInDeploymentSet *bool `type:"boolean"`

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `type:"int32"`

	PrimaryIpAddress *string `type:"string"`

	ProjectName *string `type:"string"`

	Status *string `type:"string"`

	SubnetId *string `type:"string"`

	TagFilters []*TagFilterForDescribeInstancesInput `type:"list"`

	VpcId *string `type:"string"`

	ZoneId *string `type:"string"`
}

type DescribeInstanceReqOptionForDescribeInstancesInput struct {
	_ struct{} `type:"structure"`

	NeedEipInfo *bool `type:"boolean"`

	NeedInstanceTypeInfo *bool `type:"boolean"`

	NeedNetworkInfo *bool `type:"boolean"`

	NeedTradeInfo *bool `type:"boolean"`

	NeedVolumeInfo *bool `type:"boolean"`
}

type FieldMaskForDescribeInstancesInput struct {
	_ struct{} `type:"structure"`

	Paths *string `type:"string"`
}

type TagFilterForDescribeInstancesInput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Values []*string `type:"list"`
}

type DescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	Metadata *response.ResponseMetadata

	Instances []*InstanceForDescribeInstancesOutput `type:"list"`

	NextToken *string `type:"string"`

	PageNumber *int32 `type:"int32"`

	PageSize *int32 `type:"int32"`

	TotalCount *int32 `type:"int32"`
}

type InstanceForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	AccountId *string `type:"string"`

	CpuOptions *CpuOptionsForDescribeInstancesOutput `type:"structure"`

	Cpus *int32 `type:"int32"`

	CreatedAt *string `type:"string"`

	DeploymentSetGroupNumber *int32 `type:"int32"`

	DeploymentSetId *string `type:"string"`

	Description *string `type:"string"`

	EipAddress *EipAddressForDescribeInstancesOutput `type:"structure"`

	ExpiredAt *string `type:"string"`

	HostName *string `type:"string"`

	Hostname *string `type:"string"`

	HpcClusterId *string `type:"string"`

	Id *string `type:"string"`

	ImageId *string `type:"string"`

	InstanceChargeType *string `type:"string"`

	InstanceId *string `type:"string"`

	InstanceName *string `type:"string"`

	InstanceType *InstanceTypeForDescribeInstancesOutput `type:"structure"`

	InstanceTypeId *string `type:"string"`

	KeyPairId *string `type:"string"`

	KeyPairName *string `type:"string"`

	LocalVolumes []*LocalVolumeForDescribeInstancesOutput `type:"list"`

	MemorySize *int32 `type:"int32"`

	NetworkInterfaces []*NetworkInterfaceForDescribeInstancesOutput `type:"list"`

	OsName *string `type:"string"`

	OsType *string `type:"string"`

	OverdueAt *string `type:"string"`

	OverdueReclaimedAt *string `type:"string"`

	ProjectName *string `type:"string"`

	RdmaIpAddresses []*string `type:"list"`

	ReclaimedAt *string `type:"string"`

	RenewType *int32 `type:"int32"`

	SpotStrategy *string `type:"string"`

	Status *string `type:"string"`

	StoppedMode *string `type:"string"`

	Tags []*TagForDescribeInstancesOutput `type:"list"`

	TradeStatus *int32 `type:"int32"`

	UpdatedAt *string `type:"string"`

	UserData *string `type:"string"`

	Uuid *string `type:"string"`

	Volumes []*VolumeForDescribeInstancesOutput `type:"list"`

	VpcId *string `type:"string"`

	ZoneId *string `type:"string"`
}

type CpuOptionsForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	CoreCount *int32 `type:"int32"`

	ThreadsPerCore *int32 `type:"int32"`
}

type EipAddressForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	AllocationId *string `type:"string"`

	Bandwidth *int32 `type:"int32"`

	IpAddress *string `type:"string"`
}

type InstanceTypeForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	Architecture *string `type:"string"`

	Cpu *int32 `type:"int32"`

	Id *string `type:"string"`

	InstanceTypeFamily *string `type:"string"`

	Mem *int32 `type:"int32"`

	NetKppsQuota *int32 `type:"int32"`

	NetMbpsQuota *int32 `type:"int32"`

	NetSessionQuota *int32 `type:"int32"`

	NetworkInterfaceNumQuota *int32 `type:"int32"`

	PrivateIpQuota *int32 `type:"int32"`

	VolumeTypes []*string `type:"list"`
}

type LocalVolumeForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	Count *int32 `type:"int32"`

	Size *int32 `type:"int32"`

	VolumeType *string `type:"string"`
}

type NetworkInterfaceForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	Ipv6Addresses []*string `type:"list"`

	MacAddress *string `type:"string"`

	NetworkInterfaceId *string `type:"string"`

	PrimaryIpAddress *string `type:"string"`

	PrivateIpAddresses []*string `type:"list"`

	SubnetId *string `type:"string"`

	Type *string `type:"string"`

	VpcId *string `type:"string"`
}

type TagForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	Key *string `type:"string"`

	Value *string `type:"string"`
}

type VolumeForDescribeInstancesOutput struct {
	_ struct{} `type:"structure"`

	DeleteWithInstance *bool `type:"boolean"`

	ImageId *string `type:"string"`

	Kind *string `type:"string"`

	Size *string `type:"string"`

	Status *string `type:"string"`

	VolumeId *string `type:"string"`

	VolumeName *string `type:"string"`

	VolumeType *string `type:"string"`
}
