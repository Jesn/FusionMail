package handler

import (
	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/internal/service"
	"fusionmail/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var groupHandlerLog = logger.NewWithModule("GroupHandler")

type GroupHandler struct {
	groupService service.GroupService
}

func NewGroupHandler(groupService service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type AssignAccountRequest struct {
	GroupID *int64 `json:"group_id"`
}

type BatchAssignRequest struct {
	AccountUIDs []string `json:"account_uids" binding:"required"`
	GroupID     *int64   `json:"group_id"`
}

type ReorderGroupsRequest struct {
	GroupIDs []int64 `json:"group_ids" binding:"required"`
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	group, err := h.groupService.CreateGroup(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		if err == model.ErrGroupNameRequired {
			dto.BadRequestResponse(c, err.Error())
			return
		}
		if err == model.ErrGroupNameExists {
			dto.ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		groupHandlerLog.Error("创建分组失败: %v", err)
		dto.InternalServerErrorResponse(c, "创建分组失败")
		return
	}
	dto.SuccessWithMessage(c, group, "创建成功")
}

func (h *GroupHandler) GetGroups(c *gin.Context) {
	// 使用带统计信息的新方法
	response, err := h.groupService.GetGroupsWithStats(c.Request.Context())
	if err != nil {
		groupHandlerLog.Error("获取分组列表失败: %v", err)
		dto.InternalServerErrorResponse(c, "获取分组列表失败")
		return
	}
	dto.SuccessResponse(c, response)
}

func (h *GroupHandler) GetGroupByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的分组 ID")
		return
	}
	group, err := h.groupService.GetGroupByID(c.Request.Context(), id)
	if err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, err.Error())
			return
		}
		groupHandlerLog.Error("获取分组详情失败: %v", err)
		dto.InternalServerErrorResponse(c, "获取分组详情失败")
		return
	}
	dto.SuccessResponse(c, group)
}

func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的分组 ID")
		return
	}
	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	group, err := h.groupService.UpdateGroup(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, err.Error())
			return
		}
		if err == model.ErrGroupNameExists {
			dto.ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		groupHandlerLog.Error("更新分组失败: %v", err)
		dto.InternalServerErrorResponse(c, "更新分组失败")
		return
	}
	dto.SuccessWithMessage(c, group, "更新成功")
}

func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		dto.BadRequestResponse(c, "无效的分组 ID")
		return
	}
	if err := h.groupService.DeleteGroup(c.Request.Context(), id); err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, err.Error())
			return
		}
		groupHandlerLog.Error("删除分组失败: %v", err)
		dto.InternalServerErrorResponse(c, "删除分组失败")
		return
	}
	dto.SuccessWithMessage(c, nil, "删除成功")
}

func (h *GroupHandler) AssignAccountToGroup(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		dto.BadRequestResponse(c, "账号 UID 不能为空")
		return
	}
	var req AssignAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	if err := h.groupService.AssignAccountToGroup(c.Request.Context(), uid, req.GroupID); err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, "分组不存在")
			return
		}
		if apiErr, ok := err.(*dto.APIError); ok && apiErr.Code == dto.ErrAccountNotFound {
			dto.NotFoundResponse(c, "账号不存在")
			return
		}
		groupHandlerLog.Error("分配账号到分组失败: %v", err)
		dto.InternalServerErrorResponse(c, "分配账号到分组失败")
		return
	}
	dto.SuccessWithMessage(c, nil, "分配成功")
}

func (h *GroupHandler) BatchAssignAccounts(c *gin.Context) {
	var req BatchAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	if len(req.AccountUIDs) == 0 {
		dto.BadRequestResponse(c, "账号列表不能为空")
		return
	}
	if err := h.groupService.BatchAssignAccounts(c.Request.Context(), req.AccountUIDs, req.GroupID); err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, "分组不存在")
			return
		}
		groupHandlerLog.Error("批量分配账号失败: %v", err)
		dto.InternalServerErrorResponse(c, "批量分配账号失败")
		return
	}
	dto.SuccessWithMessage(c, gin.H{"count": len(req.AccountUIDs)}, "批量分配成功")
}

func (h *GroupHandler) ReorderGroups(c *gin.Context) {
	var req ReorderGroupsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	if len(req.GroupIDs) == 0 {
		dto.BadRequestResponse(c, "分组列表不能为空")
		return
	}
	if err := h.groupService.ReorderGroups(c.Request.Context(), req.GroupIDs); err != nil {
		if err == model.ErrGroupNotFound {
			dto.NotFoundResponse(c, "分组不存在")
			return
		}
		groupHandlerLog.Error("重排序分组失败: %v", err)
		dto.InternalServerErrorResponse(c, "重排序分组失败")
		return
	}
	dto.SuccessWithMessage(c, nil, "重排序成功")
}

func (h *GroupHandler) GetUngroupedAccounts(c *gin.Context) {
	accounts, err := h.groupService.GetUngroupedAccounts(c.Request.Context())
	if err != nil {
		groupHandlerLog.Error("获取未分组账号失败: %v", err)
		dto.InternalServerErrorResponse(c, "获取未分组账号失败")
		return
	}
	dto.SuccessResponse(c, accounts)
}
