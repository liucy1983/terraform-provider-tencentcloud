package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/pkg/errors"
	SDKErrors "github.com/liucy1983/tencentcloud-sdk-go/tencentcloud/common/errors"
	sqlserver "github.com/liucy1983/tencentcloud-sdk-go/tencentcloud/sqlserver/v20180328"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/connectivity"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/internal/helper"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/ratelimit"
)

type SqlserverService struct {
	client *connectivity.TencentCloudClient
}

func (me *SqlserverService) DescribeZones(ctx context.Context) (zoneInfoList []*sqlserver.ZoneInfo, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeZonesRequest()

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	var response *sqlserver.DescribeZonesResponse
	err := resource.Retry(readRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		result, e := me.client.UseSqlserverClient().DescribeZones(request)
		if e != nil {
			log.Printf("[CRITAL]%s DescribeZones fail, reason:%s\n", logId, e.Error())
			return retryError(e)
		}
		response = result
		return nil
	})
	if err != nil {
		errRet = err
		return
	}
	if response != nil && response.Response != nil {
		zoneInfoList = response.Response.ZoneSet
	}
	return
}

func (me *SqlserverService) DescribeProductConfig(ctx context.Context, zone string) (specInfoList []*sqlserver.SpecInfo, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeProductConfigRequest()
	request.Zone = &zone

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	var response *sqlserver.DescribeProductConfigResponse
	err := resource.Retry(readRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		result, e := me.client.UseSqlserverClient().DescribeProductConfig(request)
		if e != nil {
			log.Printf("[CRITAL]%s DescribeProductConfig fail, reason:%s\n", logId, e.Error())
			return retryError(e)
		}
		response = result
		return nil
	})
	if err != nil {
		errRet = err
		return
	}
	if response != nil && response.Response != nil {
		specInfoList = response.Response.SpecInfoList
	}
	return
}

func (me *SqlserverService) CreateSqlserverInstance(ctx context.Context, dbVersion string, chargeType string, memory int, autoRenewFlag int, projectId int, subnetId string, vpcId string, zone string, storage int, weekSet []int, startTime string, timeSpan int, multiZones bool, haType string, securityGroups []string) (instanceId string, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewCreateDBInstancesRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	request.DBVersion = &dbVersion
	request.InstanceChargeType = &chargeType
	request.Memory = helper.IntInt64(memory)
	request.Storage = helper.IntInt64(storage)
	request.SubnetId = &subnetId
	request.VpcId = &vpcId
	request.HAType = &haType
	request.MultiZones = &multiZones
	request.AutoRenewFlag = helper.IntInt64(autoRenewFlag)
	if projectId != 0 {
		request.ProjectId = helper.IntInt64(projectId)
	}

	if len(weekSet) > 0 {
		request.Weekly = make([]*int64, 0)
		for _, i := range weekSet {
			request.Weekly = append(request.Weekly, helper.IntInt64(i))
		}
	}
	if startTime != "" {
		request.StartTime = &startTime
	}
	if timeSpan != 0 {
		request.Span = helper.IntInt64(timeSpan)
	}

	request.SecurityGroupList = make([]*string, 0, len(securityGroups))
	for _, v := range securityGroups {
		request.SecurityGroupList = append(request.SecurityGroupList, &v)
	}

	request.GoodsNum = helper.IntInt64(1)
	request.Zone = &zone

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().CreateDBInstances(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}
	if len(response.Response.DealNames) == 0 {
		errRet = fmt.Errorf("TencentCloud SDK returns empty SQL Server ID")
		return
	} else if len(response.Response.DealNames) > 1 {
		errRet = fmt.Errorf("TencentCloud SDK returns more than one SQL Server ID")
		return
	}

	dealId := *response.Response.DealNames[0]
	instanceId, err = me.GetInfoFromDeal(ctx, dealId)
	if err != nil {
		errRet = err
	}
	return
}

func (me *SqlserverService) ModifySqlserverInstanceName(ctx context.Context, instanceId string, name string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyDBInstanceNameRequest()
	request.InstanceId = &instanceId
	request.InstanceName = &name
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().ModifyDBInstanceName(request)
	return err
}

func (me *SqlserverService) ModifySqlserverInstanceProjectId(ctx context.Context, instanceId string, projectId int) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyDBInstanceProjectRequest()
	request.InstanceIdSet = []*string{&instanceId}
	request.ProjectId = helper.IntInt64(projectId)
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().ModifyDBInstanceProject(request)
	return err
}

func (me *SqlserverService) UpgradeSqlserverInstance(ctx context.Context, instanceId string, memory int, storage int) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewUpgradeDBInstanceRequest()
	request.InstanceId = &instanceId
	request.Memory = helper.IntInt64(memory)
	request.Storage = helper.IntInt64(storage)
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().UpgradeDBInstance(request)
	if err != nil {
		return err
	}

	//check status not expanding
	errRet = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		instance, has, err := me.DescribeSqlserverInstanceById(ctx, instanceId)
		if err != nil {
			return resource.NonRetryableError(errors.WithStack(err))
		}
		if !has {
			return resource.NonRetryableError(fmt.Errorf("cannot find SQL Server instance %s", instanceId))
		}
		if int(*instance.Status) == 9 {
			return resource.RetryableError(fmt.Errorf("expanding , SQL Server instance ID %s, status %d.... ", instanceId, *instance.Status))
		} else {
			return nil
		}
	})

	return
}

func (me *SqlserverService) RemoveSecurityGroup(ctx context.Context, instanceId string, securityGroupId string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDisassociateSecurityGroupsRequest()
	request.InstanceIdSet = []*string{&instanceId}
	request.SecurityGroupId = &securityGroupId

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().DisassociateSecurityGroups(request)

	return err
}

func (me *SqlserverService) AddSecurityGroup(ctx context.Context, instanceId string, securityGroupId string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewAssociateSecurityGroupsRequest()
	request.InstanceIdSet = []*string{&instanceId}
	request.SecurityGroupId = &securityGroupId

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().AssociateSecurityGroups(request)
	time.Sleep(10 * time.Second)
	return err
}

func (me *SqlserverService) ModifySqlserverInstanceMaintenanceSpan(ctx context.Context, instanceId string, weekSet []int, startTime string, timeSpan int) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyMaintenanceSpanRequest()
	request.InstanceId = &instanceId
	if len(weekSet) > 0 {
		request.Weekly = make([]*int64, 0)
		for _, i := range weekSet {
			request.Weekly = append(request.Weekly, helper.IntInt64(i))
		}
	}
	if startTime != "" {
		request.StartTime = &startTime
	}
	if timeSpan != 0 {
		request.Span = helper.IntUint64(timeSpan)
	}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().ModifyMaintenanceSpan(request)

	return err
}

func (me *SqlserverService) TerminateSqlserverInstance(ctx context.Context, instanceId string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewTerminateDBInstanceRequest()
	request.InstanceIdSet = []*string{&instanceId}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().TerminateDBInstance(request)
	return err
}

func (me *SqlserverService) DeleteSqlserverInstance(ctx context.Context, instanceId string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDeleteDBInstanceRequest()
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().DeleteDBInstance(request)
	return err
}

func (me *SqlserverService) DescribeSqlserverInstances(ctx context.Context, instanceId string, projectId int, vpcId string, subnetId string, netType int) (instanceList []*sqlserver.DBInstance, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeDBInstancesRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	if instanceId != "" {
		request.InstanceIdSet = []*string{&instanceId}
	}
	if projectId != -1 {
		request.ProjectId = helper.IntUint64(projectId)
	}
	if subnetId != "" && netType != BASIC_NETWORK {
		request.SubnetId = &subnetId
	}
	if vpcId != "" && netType != BASIC_NETWORK {
		request.VpcId = &vpcId
	}

	if netType == BASIC_NETWORK {
		//basic network
		request.VpcId = helper.String("")
		request.SubnetId = helper.String("")
	}
	var offset, limit int64 = 0, 20

	request.Offset = &offset
	request.Limit = &limit

	for {
		ratelimit.Check(request.GetAction())
		response, err := me.client.UseSqlserverClient().DescribeDBInstances(request)
		if err != nil {
			errRet = err
			return
		}
		if response == nil || response.Response == nil {
			errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
			return
		}
		instanceList = append(instanceList, response.Response.DBInstances...)
		if len(response.Response.DBInstances) < int(limit) {
			return
		}
		offset += limit
	}
}

func (me *SqlserverService) DescribeSqlserverInstanceById(ctx context.Context, instanceId string) (instance *sqlserver.DBInstance, has bool, errRet error) {
	instanceList, err := me.DescribeSqlserverInstances(ctx, instanceId, -1, "", "", 1)
	if err != nil {
		errRet = err
		return
	}
	if len(instanceList) == 0 {
		return
	} else if len(instanceList) > 1 {
		errRet = fmt.Errorf("[DescribeDBInstanceById]SDK returns more than one instance with instanceId %s", instanceId)
	}

	instance = instanceList[0]
	if instance != nil && *instance.Status != 8 && *instance.Status != 4 && *instance.Status != 6 {
		has = true
	}
	return
}

func (me *SqlserverService) DescribeMaintenanceSpan(ctx context.Context, instanceId string) (weekSet []int, startTime string, timeSpan int, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeMaintenanceSpanRequest()
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()
	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().DescribeMaintenanceSpan(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}

	for _, v := range response.Response.Weekly {
		weekSet = append(weekSet, int(*v))
	}
	startTime = *response.Response.StartTime
	timeSpan = int(*response.Response.Span)

	return
}

func (me *SqlserverService) DescribeInstanceSecurityGroups(ctx context.Context, instanceId string) (securityGroups []string, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeDBSecurityGroupsRequest()
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()
	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().DescribeDBSecurityGroups(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}
	securityGroups = make([]string, 0, len(response.Response.SecurityGroupSet))
	for _, v := range response.Response.SecurityGroupSet {
		securityGroups = append(securityGroups, *v.SecurityGroupId)
	}

	return
}

func (me *SqlserverService) DescribeSqlserverBackups(ctx context.Context, instanceId string, startTime string, endTime string) (backupList []*sqlserver.Backup, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeBackupsRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	request.InstanceId = &instanceId
	request.StartTime = &startTime
	request.EndTime = &endTime

	var offset, limit int64 = 0, 20

	request.Offset = &offset
	request.Limit = &limit

	for {
		ratelimit.Check(request.GetAction())
		response, err := me.client.UseSqlserverClient().DescribeBackups(request)
		if err != nil {
			errRet = err
			return
		}
		if response == nil || response.Response == nil {
			errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
			return
		}
		backupList = append(backupList, response.Response.Backups...)
		if len(response.Response.Backups) < int(limit) {
			return
		}
		offset += limit
	}
}

func (me *SqlserverService) DescribeReadonlyGroupList(ctx context.Context, instanceId string) (groupList []*sqlserver.ReadOnlyGroup, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeReadOnlyGroupListRequest()
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()
	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().DescribeReadOnlyGroupList(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}

	groupList = response.Response.ReadOnlyGroupSet

	return
}

func (me *SqlserverService) CreateSqlserverReadonlyInstance(ctx context.Context, masterInstanceId string, subnetId string, vpcId string, chargeType string, memory int, zone string, storage int, readonlyGroupType int, readonlyGroupId string, forceUpgrade bool, securityGroups []string) (instanceId string, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewCreateReadOnlyDBInstancesRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	request.InstanceId = &masterInstanceId
	request.InstanceChargeType = &chargeType
	request.Memory = helper.IntInt64(memory)
	request.Storage = helper.IntInt64(storage)
	request.SubnetId = &subnetId
	request.VpcId = &vpcId
	request.GoodsNum = helper.IntInt64(1)

	request.ReadOnlyGroupType = helper.IntInt64(readonlyGroupType)
	if readonlyGroupId != "" {
		request.ReadOnlyGroupId = &readonlyGroupId
	}

	if forceUpgrade {
		request.ReadOnlyGroupForcedUpgrade = helper.BoolToInt64Ptr(forceUpgrade)
	}
	request.GoodsNum = helper.IntInt64(1)
	request.Zone = &zone
	request.SecurityGroupList = make([]*string, 0, len(securityGroups))
	for _, v := range securityGroups {
		request.SecurityGroupList = append(request.SecurityGroupList, &v)
	}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().CreateReadOnlyDBInstances(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}
	if len(response.Response.DealNames) == 0 {
		errRet = fmt.Errorf("TencentCloud SDK returns empty SQL Server ID")
		return
	} else if len(response.Response.DealNames) > 1 {
		errRet = fmt.Errorf("TencentCloud SDK returns more than one SQL Server ID")
		return
	}

	dealId := *response.Response.DealNames[0]

	instanceId, err = me.GetInfoFromDeal(ctx, dealId)

	if err != nil {
		errRet = err
	}
	return
}

func (me *SqlserverService) DescribeReadonlyGroupListByReadonlyInstanceId(ctx context.Context, instanceId string) (readonlyGroupId string, masterInstanceId string, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeReadOnlyGroupByReadOnlyInstanceRequest()
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()
	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().DescribeReadOnlyGroupByReadOnlyInstance(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}

	readonlyGroupId = *response.Response.ReadOnlyGroupId
	masterInstanceId = *response.Response.MasterInstanceId
	return
}

func (me *SqlserverService) CreateSqlserverAccount(ctx context.Context, instanceId string, userName string, password string, remark string, isAdmin bool) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewCreateAccountRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	request.InstanceId = &instanceId
	account := sqlserver.AccountCreateInfo{UserName: &userName, Password: &password, IsAdmin: &isAdmin, Remark: &remark}
	request.Accounts = []*sqlserver.AccountCreateInfo{&account}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().CreateAccount(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}

	flowId := *response.Response.FlowId
	err = me.WaitForTaskFinish(ctx, flowId)
	if err != nil {
		errRet = err
	}
	return
}

func (me *SqlserverService) DescribeSqlserverAccounts(ctx context.Context, instanceId string) (accounts []*sqlserver.AccountDetail, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeAccountsRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	request.InstanceId = &instanceId

	var offset, limit uint64 = 0, 20

	request.Offset = &offset
	request.Limit = &limit

	for {
		ratelimit.Check(request.GetAction())
		response, err := me.client.UseSqlserverClient().DescribeAccounts(request)
		if err != nil {
			ee, ok := err.(*SDKErrors.TencentCloudSDKError)
			if !ok || ee.Code != "ResourceNotFound.InstanceNotFound" {
				errRet = err
				return
			}

			return

		}
		if response == nil || response.Response == nil {
			errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
			return
		}
		accounts = append(accounts, response.Response.Accounts...)
		if len(response.Response.Accounts) < int(limit) {
			return
		}
		offset += limit
	}
}

func (me *SqlserverService) DescribeSqlserverAccountById(ctx context.Context, instanceId string, userName string) (account *sqlserver.AccountDetail, has bool, errRet error) {
	accountList, err := me.DescribeSqlserverAccounts(ctx, instanceId)
	if err != nil {
		errRet = err
		return
	}
	if len(accountList) == 0 {
		return
	}

	for _, v := range accountList {
		if *v.Name == userName {
			account = v
			has = true
			return
		}
	}
	return
}

func (me *SqlserverService) ModifySqlserverAccountRemark(ctx context.Context, instanceId string, userName string, remark string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyAccountRemarkRequest()
	request.InstanceId = &instanceId
	request.Accounts = []*sqlserver.AccountRemark{{UserName: &userName, Remark: &remark}}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().ModifyAccountRemark(request)
	return err
}

func (me *SqlserverService) ResetSqlserverAccountPassword(ctx context.Context, instanceId string, userName string, password string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewResetAccountPasswordRequest()
	request.InstanceId = &instanceId
	request.Accounts = []*sqlserver.AccountPassword{{UserName: &userName, Password: &password}}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().ResetAccountPassword(request)
	if err != nil {
		errRet = err
		return
	}

	//check status not resetting
	errRet = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		instance, has, err := me.DescribeSqlserverAccountById(ctx, instanceId, userName)
		if err != nil {
			return resource.NonRetryableError(errors.WithStack(err))
		}
		if !has {
			return resource.NonRetryableError(fmt.Errorf("cannot find SQL Server account %s%s%s", instanceId, FILED_SP, userName))
		}
		if int(*instance.Status) == 4 {
			return resource.RetryableError(fmt.Errorf("resetting , SQL Server instance ID %s, name %s, status %d.... ", instanceId, userName, *instance.Status))
		} else {
			return nil
		}
	})

	return
}

func (me *SqlserverService) DeleteSqlserverAccount(ctx context.Context, instanceId string, userName string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDeleteAccountRequest()
	request.UserNames = []*string{&userName}
	request.InstanceId = &instanceId
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	_, err := me.client.UseSqlserverClient().DeleteAccount(request)
	if err != nil {
		ee, ok := err.(*SDKErrors.TencentCloudSDKError)
		if !ok || ee.Code != "ResourceNotFound.InstanceNotFound" {
			errRet = err
			return
		}
		return
	}

	//check status not deleting
	errRet = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		instance, has, err := me.DescribeSqlserverAccountById(ctx, instanceId, userName)
		if err != nil {
			return resource.NonRetryableError(errors.WithStack(err))
		}
		if !has {
			return nil
		}
		if int(*instance.Status) == -1 {
			return resource.RetryableError(fmt.Errorf("deleting , SQL Server instance ID %s, name %s, status %d.... ", instanceId, userName, *instance.Status))
		} else {
			return resource.NonRetryableError(fmt.Errorf("invalid, SQL Server instance ID %s, name %s, status %d...", instanceId, userName, *instance.Status))
		}
	})

	return
}

func (me *SqlserverService) ModifyAccountDBAttachment(ctx context.Context, instanceId, accountName, dbName, privilege string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyAccountPrivilegeRequest()
	request.InstanceId = &instanceId
	request.Accounts = []*sqlserver.AccountPrivilegeModifyInfo{{UserName: &accountName, DBPrivileges: []*sqlserver.DBPrivilegeModifyInfo{{DBName: &dbName, Privilege: &privilege}}}}

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	//check account exists
	_, has, err := me.DescribeSqlserverAccountById(ctx, instanceId, accountName)

	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("SQL Server account %s , instance ID %s is not exist", accountName, instanceId)
	}

	//check db exists
	_, has, err = me.DescribeDBDetailsById(ctx, fmt.Sprintf("%s%s%s", instanceId, FILED_SP, dbName))
	if err != nil {
		return err
	}

	if !has {
		return fmt.Errorf("SQL Server DB %s , instance ID %s is not exist", dbName, instanceId)
	}

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().ModifyAccountPrivilege(request)
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK return nil response, %s", request.GetAction())
		return
	}

	if err != nil {
		return err
	}

	flowId := int64(*response.Response.FlowId)
	err = me.WaitForTaskFinish(ctx, flowId)
	return err
}

func (me *SqlserverService) DescribeAccountDBAttachments(ctx context.Context, instanceId, accountName, dbName string) (attachments []map[string]string, errRet error) {

	if accountName != "" {
		//check account exists
		accounts, has, err := me.DescribeSqlserverAccountById(ctx, instanceId, accountName)
		if err != nil {
			errRet = err
			return
		}
		if !has {
			return
		}

		for _, v := range accounts.Dbs {

			if (dbName != "" && *v.DBName == dbName) || dbName == "" {
				mapping := make(map[string]string)
				mapping["db_name"] = *v.DBName
				mapping["account_name"] = accountName
				mapping["privilege"] = *v.Privilege
				attachments = append(attachments, mapping)
			}
		}
		return
	} else {
		dbInfos, err := me.DescribeDBsOfInstance(ctx, instanceId)
		if err != nil {
			errRet = err
			return
		}
		if len(dbInfos) == 0 {
			return
		}
		for _, v := range dbInfos {
			if (dbName != "" && *v.Name == dbName) || dbName == "" {
				for _, vv := range v.Accounts {
					mapping := make(map[string]string)
					mapping["db_name"] = *v.Name
					mapping["account_name"] = *vv.UserName
					mapping["privilege"] = *vv.Privilege
					attachments = append(attachments, mapping)
				}
			}
		}
		return
	}
}

func (me *SqlserverService) DescribeAccountDBAttachmentById(ctx context.Context, instanceId, accountName, dbName string) (attachment map[string]string, has bool, errRet error) {
	attachments, err := me.DescribeAccountDBAttachments(ctx, instanceId, accountName, dbName)
	if err != nil {
		errRet = err
		return
	}

	if len(attachments) == 0 {
		return
	}

	attachment = attachments[0]
	has = true
	return
}

func (me *SqlserverService) GetInfoFromDeal(ctx context.Context, dealId string) (instanceId string, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeOrdersRequest()
	request.DealNames = []*string{&dealId}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	ratelimit.Check(request.GetAction())
	response, err := me.client.UseSqlserverClient().DescribeOrders(request)
	if err != nil {
		errRet = err
		return
	}
	if response == nil || response.Response == nil {
		errRet = fmt.Errorf("TencentCloud SDK returns nil response, %s", request.GetAction())
		return
	}
	if len(response.Response.Deals) == 0 {
		errRet = fmt.Errorf("TencentCloud SDK returns empty deal")
		return
	} else if len(response.Response.Deals) > 1 {
		errRet = fmt.Errorf("TencentCloud SDK returns more than one deal")
		return
	}
	instanceId = *response.Response.Deals[0].InstanceIdSet[0]
	flowId := *response.Response.Deals[0].FlowId
	err = me.WaitForTaskFinish(ctx, flowId)
	if err != nil {
		errRet = err
	}
	return
}

func (me *SqlserverService) WaitForTaskFinish(ctx context.Context, flowId int64) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeFlowStatusRequest()
	request.FlowId = &flowId

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	errRet = resource.Retry(4*writeRetryTimeout, func() *resource.RetryError {
		taskResponse, err := me.client.UseSqlserverClient().DescribeFlowStatus(request)
		log.Printf("This is the time out interface, thanks2")
		ratelimit.Check(request.GetAction())
		if err != nil {
			return resource.NonRetryableError(errors.WithStack(err))
		}
		if *taskResponse.Response.Status == int64(SQLSERVER_TASK_RUNNING) {
			return resource.RetryableError(errors.WithStack(fmt.Errorf("SQLSERVER task status is %d(task running), requestId is %s", *taskResponse.Response.Status, *taskResponse.Response.RequestId)))
		} else if *taskResponse.Response.Status == int64(SQLSERVER_TASK_FAIL) {
			return resource.NonRetryableError(errors.WithStack(fmt.Errorf("SQLSERVER task status is %d(failed), requestId is %s", *taskResponse.Response.Status, *taskResponse.Response.RequestId)))
		}
		return nil
	})
	return
}

func (me *SqlserverService) CreateSqlserverDB(ctx context.Context, instanceID string, dbname string, charset string, remark string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewCreateDBRequest()

	// set instance id
	request.InstanceId = &instanceID
	// set DBs
	var dbCreateInfo = sqlserver.DBCreateInfo{
		DBName:  &dbname,
		Charset: &charset,
		Remark:  &remark,
	}
	var dbInfoList []*sqlserver.DBCreateInfo
	dbInfoList = append(dbInfoList, &dbCreateInfo)
	request.DBs = dbInfoList

	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]\n",
				logId, request.GetAction(), request.ToJsonString(), errRet.Error())
		}
	}()

	var response *sqlserver.CreateDBResponse
	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		result, e := me.client.UseSqlserverClient().CreateDB(request)
		if e != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason:%s\n", logId, request.GetAction(), e.Error())
			return retryError(e)
		}
		response = result
		return nil
	})
	if err != nil {
		errRet = err
		return
	}

	if response != nil && response.Response != nil && *response.Response.FlowId != 0 {
		return me.WaitForTaskFinish(ctx, *response.Response.FlowId)
	}
	return
}

func (me *SqlserverService) DescribeDBsOfInstance(ctx context.Context, instanceId string) (instanceDBList []*sqlserver.DBDetail, errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDescribeDBsRequest()
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	if instanceId != "" {
		request.InstanceIdSet = []*string{&instanceId}
	}
	var offset, limit uint64 = SQLSERVER_DEFAULT_OFFSET, SQLSERVER_DEFAULT_LIMIT

	request.Offset = &offset
	request.Limit = &limit

	for {
		var response *sqlserver.DescribeDBsResponse
		err := resource.Retry(readRetryTimeout, func() *resource.RetryError {
			ratelimit.Check(request.GetAction())
			result, e := me.client.UseSqlserverClient().DescribeDBs(request)
			if e != nil {
				log.Printf("[CRITAL]%s api[%s] fail, reason:%s\n", logId, request.GetAction(), e.Error())
				return retryError(e)
			}
			response = result
			return nil
		})
		if err != nil {
			errRet = err
			return
		}
		if response == nil || response.Response == nil {
			errRet = fmt.Errorf("TencentCloud SDK returns nil response for api[%s]", request.GetAction())
			return
		}
		if len(response.Response.DBInstances) == 0 {
			return
		} else if len(response.Response.DBInstances) > 1 {
			errRet = fmt.Errorf("[CRITAL]%s api[%s] returned multiple DB lists for one instance", logId, request.GetAction())
			return
		}
		instanceDBList = append(instanceDBList, response.Response.DBInstances[0].DBDetails...)
		if len(response.Response.DBInstances) < int(limit) {
			return
		}
		offset += limit
	}
}

func (me *SqlserverService) DescribeDBDetailsById(ctx context.Context, dbId string) (dbInfo *sqlserver.DBDetail, has bool, errRet error) {
	idItem := strings.Split(dbId, FILED_SP)
	if len(idItem) < 2 {
		errRet = fmt.Errorf("broken ID of SQLServer DB %s", dbId)
		return
	}
	instanceId := idItem[0]
	dbName := idItem[1]

	instanceDBList, err := me.DescribeDBsOfInstance(ctx, instanceId)
	if err != nil {
		errRet = err
		return
	}
	if len(instanceDBList) == 0 {
		return
	}

	for _, dbDetail := range instanceDBList {
		if *dbDetail.Name == dbName {
			dbInfo = dbDetail
			if *dbDetail.Status != SQLSERVER_DB_DELETING {
				has = true
			}
			break
		}
	}
	return
}

func (me *SqlserverService) ModifySqlserverDBRemark(ctx context.Context, instanceId string, dbName string, remark string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewModifyDBRemarkRequest()
	request.InstanceId = &instanceId
	request.DBRemarks = []*sqlserver.DBRemark{{Name: &dbName, Remark: &remark}}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		_, e := me.client.UseSqlserverClient().ModifyDBRemark(request)
		if e != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason:%s\n", logId, request.GetAction(), e.Error())
			return retryError(e)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return
}

func (me *SqlserverService) DeleteSqlserverDB(ctx context.Context, instanceId string, name string) (errRet error) {
	logId := getLogId(ctx)
	request := sqlserver.NewDeleteDBRequest()
	request.InstanceId = &instanceId
	request.Names = []*string{&name}
	defer func() {
		if errRet != nil {
			log.Printf("[CRITAL]%s api[%s] fail, reason[%s]", logId, request.GetAction(), errRet.Error())
		}
	}()

	var response *sqlserver.DeleteDBResponse
	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		result, e := me.client.UseSqlserverClient().DeleteDB(request)
		if e != nil {
			log.Printf("[CRITAL]%s %s fail, reason:%s\n", logId, request.GetAction(), e.Error())
			return retryError(e)
		}
		response = result
		return nil
	})
	if err != nil {
		return err
	}

	if response != nil && response.Response != nil {
		return me.WaitForTaskFinish(ctx, *response.Response.FlowId)
	}
	return
}
