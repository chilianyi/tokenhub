package controller

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetAllOrganizations 部门列表（分页）
func GetAllOrganizations(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	orgs, total, err := model.GetOrganizations(c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		organizationAPIError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orgs)
	common.ApiSuccess(c, pageInfo)
}

// GetOrganizationMeta 部门详情
func GetOrganizationMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		organizationAPIError(c, err)
		return
	}
	org, err := model.GetOrganizationById(id)
	if err != nil {
		organizationAPIError(c, err)
		return
	}
	common.ApiSuccess(c, org)
}

// CreateOrganization 新建部门
func CreateOrganization(c *gin.Context) {
	var org model.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		organizationAPIError(c, err)
		return
	}
	if err := org.Insert(); err != nil {
		organizationAPIError(c, err)
		return
	}
	recordManageAudit(c, "organization.create", map[string]any{"organization_id": org.Id, "name": org.Name, "quota": org.Quota})
	common.ApiSuccess(c, &org)
}

// UpdateOrganization 更新部门。
//
// 额度池调低到已分配额度以下时必须拒绝：否则不变式 Allocated <= Quota 被破坏，
// 后续所有分配都会被误判为超额。
func UpdateOrganization(c *gin.Context) {
	var org model.Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		organizationAPIError(c, err)
		return
	}
	if org.Id == 0 {
		common.ApiErrorT(c, "Missing organization ID")
		return
	}
	current, err := model.GetOrganizationById(org.Id)
	if err != nil {
		organizationAPIError(c, err)
		return
	}
	if org.Quota < current.Allocated {
		common.ApiErrorT(c, fmt.Sprintf(
			"Quota cannot be lower than the amount already allocated to members (%d)", current.Allocated))
		return
	}
	if err := org.Update(); err != nil {
		organizationAPIError(c, err)
		return
	}
	recordManageAudit(c, "organization.update", map[string]any{
		"organization_id": org.Id,
		"name":            org.Name,
		"quota_before":    current.Quota,
		"quota_after":     org.Quota,
	})
	common.ApiSuccess(c, &org)
}

// DeleteOrganization 删除部门（软删除，成员归属清空）
func DeleteOrganization(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		organizationAPIError(c, err)
		return
	}
	if err := model.DeleteOrganizationByID(id); err != nil {
		organizationAPIError(c, err)
		return
	}
	recordManageAudit(c, "organization.delete", map[string]any{"organization_id": id})
	common.ApiSuccess(c, nil)
}

func organizationAPIError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrOrgNotFound):
		c.JSON(404, gin.H{"success": false, "message": "Organization not found", "code": "ORGANIZATION_NOT_FOUND"})
	case errors.Is(err, model.ErrOrgQuotaExceeded):
		c.JSON(409, gin.H{"success": false, "message": err.Error(), "code": "ORGANIZATION_QUOTA_EXCEEDED"})
	case errors.Is(err, model.ErrOrgNameDuplicated):
		c.JSON(409, gin.H{"success": false, "message": "Organization name already exists", "code": "ORGANIZATION_NAME_CONFLICT"})
	default:
		common.ApiError(c, err)
	}
}
