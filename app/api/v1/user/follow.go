package user

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/1024casts/snake/app/api"
	"github.com/1024casts/snake/internal/service"
	"github.com/1024casts/snake/pkg/errno"
	"github.com/1024casts/snake/pkg/log"
)

// Follow 关注/取消关注
// @Summary 通过用户id关注/取消关注用户
// @Description Get an user by user id
// @Tags 用户
// @Accept  json
// @Produce  json
// @Param user_id body string true "用户id"
// @Success 200 {object} model.UserInfo "用户信息"
// @Router /users/follow [post]
func Follow(c *gin.Context) {
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("follow bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	// Get the user by the `user_id` from the database.
	_, err := service.Svc.UserSvc().GetUserByID(context.TODO(), req.UserID)
	if err != nil {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	userID := api.GetUserID(c)
	// 不能关注自己
	if userID == req.UserID {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	// 检查是否已经关注过
	isFollowed := service.Svc.UserSvc().IsFollowedUser(context.TODO(), userID, req.UserID)
	if isFollowed {
		api.SendResponse(c, errno.OK, nil)
		return
	}

	if isFollowed {
		// 取消关注
		err = service.Svc.UserSvc().CancelUserFollow(context.TODO(), userID, req.UserID)
		if err != nil {
			log.Warnf("[follow] cancel user follow err: %v", err)
			api.SendResponse(c, errno.InternalServerError, nil)
			return
		}
	} else {
		// 添加关注
		err = service.Svc.UserSvc().AddUserFollow(context.TODO(), userID, req.UserID)
		if err != nil {
			log.Warnf("[follow] add user follow err: %v", err)
			api.SendResponse(c, errno.InternalServerError, nil)
			return
		}
	}

	api.SendResponse(c, nil, nil)
}
