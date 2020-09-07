package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	ckafka "github.com/liucy1983/tencentcloud-sdk-go/tencentcloud/ckafka/v20190819"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/connectivity"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/internal/helper"
	"github.com/terraform-providers/terraform-provider-tencentcloud/tencentcloud/ratelimit"
)

type CkafkaService struct {
	client *connectivity.TencentCloudClient
}

func (me *CkafkaService) CreateUser(ctx context.Context, instanceId, user, password string) (errRet error) {
	logId := getLogId(ctx)
	request := ckafka.NewCreateUserRequest()
	request.InstanceId = &instanceId
	request.Name = &user
	request.Password = &password

	var response *ckafka.CreateUserResponse
	var err error
	err = resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().CreateUser(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if response != nil && response.Response != nil && !me.OperateStatusCheck(ctx, response.Response.Result) {
		return fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
	}
	return nil
}

func (me *CkafkaService) OperateStatusCheck(ctx context.Context, result *ckafka.JgwOperateResponse) (isSucceed bool) {
	logId := getLogId(ctx)
	if result == nil {
		log.Printf("[CRITAL]%s OperateStatusCheck fail, result is nil", logId)
		return false
	}

	if result != nil && *result.ReturnCode == "0" {
		return true
	} else {
		return false
	}
}

func (me *CkafkaService) DescribeUserByUserId(ctx context.Context, userId string) (userInfo *ckafka.User, has bool, errRet error) {
	logId := getLogId(ctx)

	items := strings.Split(userId, FILED_SP)
	if len(items) != 2 {
		errRet = fmt.Errorf("id of resource.tencentcloud_ckafka_user is wrong")
		return
	}
	instanceId, user := items[0], items[1]

	if _, has, _ = me.DescribeInstanceById(ctx, instanceId); !has {
		return
	}

	request := ckafka.NewDescribeUserRequest()
	request.InstanceId = &instanceId
	request.SearchWord = &user

	var response *ckafka.DescribeUserResponse
	var err error
	err = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().DescribeUser(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		errRet = fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
		return
	}

	if response != nil && response.Response != nil && response.Response.Result != nil && response.Response.Result.Users != nil {
		if len(response.Response.Result.Users) < 1 {
			has = false
			return
		} else if len(response.Response.Result.Users) > 1 {
			errRet = fmt.Errorf("[CRITAL]%s dumplicated users found", logId)
			return
		}

		userInfo = response.Response.Result.Users[0]
		has = true
		return
	}

	return
}

func (me *CkafkaService) ModifyPassword(ctx context.Context, instanceId, user, oldPasswd, newPasswd string) (errRet error) {
	logId := getLogId(ctx)
	request := ckafka.NewModifyPasswordRequest()
	request.InstanceId = &instanceId
	request.Name = &user
	request.Password = &oldPasswd
	request.PasswordNew = &newPasswd

	var response *ckafka.ModifyPasswordResponse
	var err error
	err = resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().ModifyPassword(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if response != nil && response.Response != nil && !me.OperateStatusCheck(ctx, response.Response.Result) {
		return fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
	}
	return nil
}

func (me *CkafkaService) DeleteUser(ctx context.Context, userId string) (errRet error) {
	logId := getLogId(ctx)

	items := strings.Split(userId, FILED_SP)
	if len(items) != 2 {
		errRet = fmt.Errorf("id of resource.tencentcloud_ckafka_user is wrong")
		return
	}
	instanceId, user := items[0], items[1]

	request := ckafka.NewDeleteUserRequest()
	request.InstanceId = &instanceId
	request.Name = &user

	var response *ckafka.DeleteUserResponse
	var err error
	err = resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().DeleteUser(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if response != nil && response.Response != nil && !me.OperateStatusCheck(ctx, response.Response.Result) {
		return fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
	}
	return nil
}

func (me *CkafkaService) DescribeUserByFilter(ctx context.Context, params map[string]interface{}) (userInfos []*ckafka.User, errRet error) {
	logId := getLogId(ctx)

	instanceId := params["instance_id"].(string)
	if _, has, _ := me.DescribeInstanceById(ctx, instanceId); !has {
		return
	}

	request := ckafka.NewDescribeUserRequest()
	var offset int64 = 0
	var pageSize = int64(CKAFKA_DESCRIBE_LIMIT)
	request.InstanceId = &instanceId
	if user, ok := params["account_name"]; ok {
		request.SearchWord = helper.String(user.(string))
	}
	request.Limit = &pageSize
	request.Offset = &offset

	userInfos = make([]*ckafka.User, 0)
	for {
		var response *ckafka.DescribeUserResponse
		var err error
		err = resource.Retry(readRetryTimeout, func() *resource.RetryError {
			ratelimit.Check(request.GetAction())
			response, err = me.client.UseCkafkaClient().DescribeUser(request)
			if err != nil {
				return retryError(err)
			}
			userInfos = append(userInfos, response.Response.Result.Users...)
			return nil
		})
		if err != nil {
			errRet = fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
			return
		} else {
			if len(response.Response.Result.Users) < CKAFKA_DESCRIBE_LIMIT {
				break
			} else {
				offset += pageSize
			}
		}
	}
	return
}

func (me *CkafkaService) CreateAcl(ctx context.Context, instanceId, resourceType, resourceName, operation, permissionType, host, principal string) (errRet error) {
	logId := getLogId(ctx)
	request := ckafka.NewCreateAclRequest()
	request.InstanceId = &instanceId
	request.ResourceType = helper.Int64(CKAFKA_ACL_RESOURCE_TYPE[resourceType])
	request.ResourceName = &resourceName
	request.Operation = helper.Int64(CKAFKA_ACL_OPERATION[operation])
	request.PermissionType = helper.Int64(CKAFKA_PERMISSION_TYPE[permissionType])
	request.Host = &host
	request.Principal = helper.String(CKAFKA_ACL_PRINCIPAL_STR + principal)

	var response *ckafka.CreateAclResponse
	var err error
	err = resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().CreateAcl(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if response != nil && response.Response != nil && !me.OperateStatusCheck(ctx, response.Response.Result) {
		return fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
	}
	return nil
}

func (me *CkafkaService) DescribeAclByFilter(ctx context.Context, params map[string]interface{}) (aclInfos []*ckafka.Acl, errRet error) {
	logId := getLogId(ctx)

	instanceId := params["instance_id"].(string)
	if _, has, _ := me.DescribeInstanceById(ctx, instanceId); !has {
		return
	}
	resourceType := params["resource_type"].(string)
	resourceName := params["resource_name"].(string)
	if resourceType == "TOPIC" {
		if _, has, _ := me.DescribeTopicById(ctx, instanceId+FILED_SP+resourceName); !has {
			return
		}
	}

	request := ckafka.NewDescribeACLRequest()
	var offset int64 = 0
	var pageSize = int64(CKAFKA_DESCRIBE_LIMIT)
	request.InstanceId = &instanceId
	request.ResourceType = helper.Int64(CKAFKA_ACL_RESOURCE_TYPE[resourceType])
	request.ResourceName = helper.String(resourceName)
	if host, ok := params["host"]; ok {
		request.SearchWord = helper.String(host.(string))
	}
	request.Limit = &pageSize
	request.Offset = &offset

	aclInfos = make([]*ckafka.Acl, 0)
	for {
		var response *ckafka.DescribeACLResponse
		var err error
		err = resource.Retry(readRetryTimeout, func() *resource.RetryError {
			ratelimit.Check(request.GetAction())
			response, err = me.client.UseCkafkaClient().DescribeACL(request)
			if err != nil {
				return retryError(err)
			}
			aclInfos = append(aclInfos, response.Response.Result.AclList...)
			return nil
		})
		if err != nil {
			errRet = fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
			return
		} else {
			if len(response.Response.Result.AclList) < CKAFKA_DESCRIBE_LIMIT {
				break
			} else {
				offset += pageSize
			}
		}
	}
	return
}

func (me *CkafkaService) DescribeAclByAclId(ctx context.Context, aclId string) (aclInfo *ckafka.Acl, has bool, errRet error) {
	// acl id is organized by "instanceId + FILED_SP + permissionType + FILED_SP + principal + FILED_SP + host + FILED_SP + operation + FILED_SP + resourceType + FILED_SP + resourceName"
	items := strings.Split(aclId, FILED_SP)
	if len(items) != 7 {
		errRet = fmt.Errorf("id of resource.tencentcloud_ckafka_acl is wrong")
		return
	}
	instanceId, permission, principal, host, operation, resourceType, resourceName := items[0], items[1], items[2], items[3], items[4], items[5], items[6]

	var params = map[string]interface{}{
		"instance_id":   instanceId,
		"resource_type": resourceType,
		"resource_name": resourceName,
		"host":          host,
	}
	aclInfos, err := me.DescribeAclByFilter(ctx, params)
	if err != nil {
		errRet = err
		return
	}
	for _, acl := range aclInfos {
		if CKAFKA_PERMISSION_TYPE_TO_STRING[*acl.PermissionType] == permission && *acl.Principal == CKAFKA_ACL_PRINCIPAL_STR+principal && CKAFKA_ACL_OPERATION_TO_STRING[*acl.Operation] == operation {
			aclInfo = acl
			has = true
			return
		}
	}
	has = false
	return
}

func (me *CkafkaService) DeleteAcl(ctx context.Context, aclId string) (errRet error) {
	logId := getLogId(ctx)

	// acl id is organized by "instanceId + FILED_SP + permissionType + FILED_SP + principal + FILED_SP + host + FILED_SP + operation + FILED_SP + resourceType + FILED_SP + resourceName"
	items := strings.Split(aclId, FILED_SP)
	if len(items) != 7 {
		errRet = fmt.Errorf("id of resource.tencentcloud_ckafka_acl is wrong")
		return
	}
	instanceId, permission, principal, host, operation, resourceType, resourceName := items[0], items[1], items[2], items[3], items[4], items[5], items[6]

	request := ckafka.NewDeleteAclRequest()
	request.InstanceId = &instanceId
	request.ResourceType = helper.Int64(CKAFKA_ACL_RESOURCE_TYPE[resourceType])
	request.ResourceName = &resourceName
	request.Operation = helper.Int64(CKAFKA_ACL_OPERATION[operation])
	request.PermissionType = helper.Int64(CKAFKA_PERMISSION_TYPE[permission])
	request.Host = &host
	request.Principal = helper.String(CKAFKA_ACL_PRINCIPAL_STR + principal)

	var response *ckafka.DeleteAclResponse
	var err error
	err = resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		response, err = me.client.UseCkafkaClient().DeleteAcl(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}
	if response != nil && response.Response != nil && !me.OperateStatusCheck(ctx, response.Response.Result) {
		return fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
	}
	return nil
}

func (me *CkafkaService) DescribeInstanceById(ctx context.Context, instanceId string) (instanceInfo *ckafka.InstanceAttributesResponse, has bool, errRet error) {
	logId := getLogId(ctx)

	request := ckafka.NewDescribeInstanceAttributesRequest()
	request.InstanceId = &instanceId
	var response *ckafka.DescribeInstanceAttributesResponse
	var err error
	err = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		response, err = me.client.UseCkafkaClient().DescribeInstanceAttributes(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})
	if err != nil {
		errRet = fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
		return
	}

	if response != nil && response.Response != nil {
		if instanceInfo = response.Response.Result; instanceInfo != nil {
			has = true
			return
		}
	}

	has = false
	return
}

func (me *CkafkaService) DescribeTopicById(ctx context.Context, topicId string) (topicInfo *ckafka.TopicAttributesResponse, has bool, errRet error) {
	logId := getLogId(ctx)

	request := ckafka.NewDescribeTopicAttributesRequest()
	items := strings.Split(topicId, FILED_SP)
	if len(items) != 2 {
		errRet = fmt.Errorf("id of resource.tencentcloud_ckafka_topic is wrong")
		return
	}
	instanceId, topicName := items[0], items[1]

	request.InstanceId = &instanceId
	request.TopicName = &topicName
	var response *ckafka.DescribeTopicAttributesResponse
	var err error
	err = resource.Retry(readRetryTimeout, func() *resource.RetryError {
		ratelimit.Check(request.GetAction())
		response, err = me.client.UseCkafkaClient().DescribeTopicAttributes(request)
		if err != nil {
			return retryError(err)
		}
		return nil
	})
	if err != nil {
		errRet = fmt.Errorf("[CRITAL]%s api[%s] fail, request body [%s], reason[%s]", logId, request.GetAction(), request.ToJsonString(), err.Error())
		return
	}

	if response != nil && response.Response != nil {
		if topicInfo = response.Response.Result; topicInfo != nil {
			has = true
			return
		}
	}

	has = false
	return
}
