package user

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"github.com/1024casts/snake/internal/idl"
	"github.com/1024casts/snake/internal/model"
	"github.com/1024casts/snake/internal/repository/user"
	v0pb "github.com/1024casts/snake/internal/rpc/user/v0"
	"github.com/1024casts/snake/pkg/auth"
	"github.com/1024casts/snake/pkg/log"
	"github.com/1024casts/snake/pkg/token"
)

const (
	// FollowStatusNormal 关注状态-正常
	FollowStatusNormal int = 1 // 正常
	// FollowStatusDelete 关注状态-删除
	FollowStatusDelete = 0 // 删除

	// MaxID 最大id
	MaxID = 0xffffffffffff
)

// 用于触发编译期的接口的合理性检查机制
// 如果userService没有实现UserService,会在编译期报错
var _ UserService = (*userService)(nil)

// UserService 用户服务接口定义
// 使用大写的UserService对外保留方法
type UserService interface {
	Register(ctx context.Context, username, email, password string) error
	EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error)
	PhoneLogin(ctx context.Context, phone int64, verifyCode int) (tokenStr string, err error)
	LoginByPhone(ctx context.Context, req *v0pb.PhoneLoginRequest) (reply *v0pb.PhoneLoginReply, err error)
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserInfoByID(ctx context.Context, id uint64) (*model.UserInfo, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	UpdateUser(ctx context.Context, id uint64, userMap map[string]interface{}) error
	BatchGetUsers(ctx context.Context, userID uint64, userIDs []uint64) ([]*model.UserInfo, error)

	// 关注
	IsFollowedUser(ctx context.Context, userID uint64, followedUID uint64) bool
	AddUserFollow(ctx context.Context, userID uint64, followedUID uint64) error
	CancelUserFollow(ctx context.Context, userID uint64, followedUID uint64) error
	GetFollowingUserList(ctx context.Context, userID uint64, lastID uint64, limit int) ([]*model.UserFollowModel, error)
	GetFollowerUserList(ctx context.Context, userID uint64, lastID uint64, limit int) ([]*model.UserFansModel, error)

	Close()
}

// userService 用小写的 service 实现接口中定义的方法
type userService struct {
	userRepo       user.BaseRepo
	userFollowRepo user.FollowRepo
	userStatRepo   user.StatRepo
}

// NewUserService 实例化一个userService
// 通过 NewService 函数初始化 Service 接口
// 依赖接口，不要依赖实现，面向接口编程
func NewUserService() UserService {
	db := model.GetDB()
	return &userService{
		userRepo:       user.NewUserRepo(db),
		userFollowRepo: user.NewUserFollowRepo(db),
		userStatRepo:   user.NewUserStatRepo(db),
	}
}

// Register 注册用户
func (srv *userService) Register(ctx context.Context, username, email, password string) error {
	pwd, err := auth.Encrypt(password)
	if err != nil {
		return errors.Wrapf(err, "encrypt password err")
	}

	u := model.UserBaseModel{
		Username:  username,
		Password:  pwd,
		Email:     email,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}
	_, err = srv.userRepo.Create(ctx, u)
	if err != nil {
		return errors.Wrapf(err, "create user")
	}
	return nil
}

// EmailLogin 邮箱登录
func (srv *userService) EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error) {
	u, err := srv.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.Wrapf(err, "get user info err by email")
	}

	// Compare the login password with the user password.
	err = auth.Compare(u.Password, password)
	if err != nil {
		return "", errors.Wrapf(err, "password compare err")
	}

	// 签发签名 Sign the json web token.
	tokenStr, err = token.Sign(ctx, token.Context{UserID: u.ID, Username: u.Username}, "")
	if err != nil {
		return "", errors.Wrapf(err, "gen token sign err")
	}

	return tokenStr, nil
}

// LoginByPhone phone login, grpc wrapper
func (srv *userService) LoginByPhone(ctx context.Context, req *v0pb.PhoneLoginRequest) (reply *v0pb.PhoneLoginReply, err error) {
	tokenStr, err := srv.PhoneLogin(ctx, req.Phone, int(req.VerifyCode))
	if err != nil {
		log.Warnf("[service.user] phone login err: %v, params: %v", err, req)
	}
	reply = &v0pb.PhoneLoginReply{
		Ret: tokenStr,
		Err: "",
	}
	return
}

// PhoneLogin 邮箱登录
func (srv *userService) PhoneLogin(ctx context.Context, phone int64, verifyCode int) (tokenStr string, err error) {
	// 如果是已经注册用户，则通过手机号获取用户信息
	u, err := srv.GetUserByPhone(ctx, phone)
	if err != nil {
		return "", errors.Wrapf(err, "[login] get u info err")
	}

	// 否则新建用户信息, 并取得用户信息
	if u.ID == 0 {
		u := model.UserBaseModel{
			Phone:    phone,
			Username: strconv.Itoa(int(phone)),
		}
		u.ID, err = srv.userRepo.Create(ctx, u)
		if err != nil {
			return "", errors.Wrapf(err, "[login] create user err")
		}
	}

	// 签发签名 Sign the json web token.
	tokenStr, err = token.Sign(ctx, token.Context{UserID: u.ID, Username: u.Username}, "")
	if err != nil {
		return "", errors.Wrapf(err, "[login] gen token sign err")
	}
	return tokenStr, nil
}

// UpdateUser update user info
func (srv *userService) UpdateUser(ctx context.Context, id uint64, userMap map[string]interface{}) error {
	err := srv.userRepo.Update(ctx, id, userMap)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID 获取单条用户信息
func (srv *userService) GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return userModel, errors.Wrapf(err, "get user info err from db by id: %d", id)
	}

	return userModel, nil
}

// GetUserInfoByID 获取组装好的用户数据
func (srv *userService) GetUserInfoByID(ctx context.Context, id uint64) (*model.UserInfo, error) {
	userInfos, err := srv.BatchGetUsers(ctx, id, []uint64{id})
	if err != nil {
		return nil, err
	}
	return userInfos[0], nil
}

// BatchGetUsers 批量获取用户信息
// 1. 处理关注和被关注状态
// 2. 获取关注和粉丝数据
func (srv *userService) BatchGetUsers(ctx context.Context, userID uint64, userIDs []uint64) ([]*model.UserInfo, error) {
	infos := make([]*model.UserInfo, 0)
	// 批量获取用户信息
	users, err := srv.userRepo.GetUsersByIds(ctx, userIDs)
	if err != nil {
		return nil, errors.Wrap(err, "[user_service] batch get user err")
	}

	// 获取当前用户信息
	curUser, err := srv.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "[user_service] get one user err")
	}

	// 保持原有id顺序
	ids := userIDs

	wg := sync.WaitGroup{}
	userList := model.UserList{
		Lock:  new(sync.Mutex),
		IDMap: make(map[uint64]*model.UserInfo, len(users)),
	}

	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	// 获取自己对关注列表的关注状态
	userFollowMap, err := srv.userFollowRepo.GetFollowByUIds(ctx, userID, userIDs)
	if err != nil {
		errChan <- err
	}

	// 获取自己对关注列表的被关注状态
	userFansMap, err := srv.userFollowRepo.GetFansByUIds(ctx, userID, userIDs)
	if err != nil {
		errChan <- err
	}

	// 获取用户统计
	userStatMap, err := srv.userStatRepo.GetUserStatByIDs(ctx, userIDs)
	if err != nil {
		errChan <- err
	}

	// 并行处理
	for _, u := range users {
		wg.Add(1)
		go func(u *model.UserBaseModel) {
			defer wg.Done()

			userList.Lock.Lock()
			defer userList.Lock.Unlock()

			isFollow := 0
			_, ok := userFollowMap[u.ID]
			if ok {
				isFollow = 1
			}

			isFollowed := 0
			_, ok = userFansMap[u.ID]
			if ok {
				isFollowed = 1
			}

			userStatMap, ok := userStatMap[u.ID]
			if !ok {
				userStatMap = nil
			}

			transInput := &idl.TransferUserInput{
				CurUser:  curUser,
				User:     u,
				UserStat: userStatMap,
				IsFollow: isFollow,
				IsFans:   isFollowed,
			}
			userInfo := idl.TransferUser(transInput)
			if err != nil {
				errChan <- err
				return
			}
			userList.IDMap[u.ID] = userInfo
		}(u)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case err := <-errChan:
		log.Warnf("[user_service] batch get user err chan: %v", err)
		return nil, err
	}

	// 根据原有id合并数据
	for _, id := range ids {
		infos = append(infos, userList.IDMap[id])
	}

	return infos, nil
}

func (srv *userService) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByPhone(ctx, phone)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by phone: %d", phone)
	}

	return userModel, nil
}

func (srv *userService) GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByEmail(ctx, email)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by email: %s", email)
	}

	return userModel, nil
}

// 获取用户关注
func (srv *userService) GetFollowUser(ctx context.Context, userID uint64, followedUID uint64) (*model.UserFollowModel, error) {
	userFollowModel := &model.UserFollowModel{}
	result := model.GetDB().
		Where("user_id=? AND followed_uid=? ", userID, followedUID).
		Find(userFollowModel)

	return userFollowModel, result.Error
}

// IsFollowedUser 是否关注过某用户
func (srv *userService) IsFollowedUser(ctx context.Context, userID uint64, followedUID uint64) bool {
	userFollowModel := &model.UserFollowModel{}
	result := model.GetDB().
		Where("user_id=? AND followed_uid=? ", userID, followedUID).
		Find(userFollowModel)

	if err := result.Error; err != nil {
		log.Warnf("[user_service] get user follow err, %v", err)
		return false
	}

	if userFollowModel.ID > 0 && userFollowModel.Status == FollowStatusNormal {
		return true
	}

	return false
}

// AddUserFollow 添加关注
func (srv *userService) AddUserFollow(ctx context.Context, userID uint64, followedUID uint64) error {
	db := model.GetDB()
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 添加到关注表
	err := srv.userFollowRepo.CreateUserFollow(ctx, tx, userID, followedUID)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "insert into user follow err")
	}

	// 添加到粉丝表
	err = srv.userFollowRepo.CreateUserFans(ctx, tx, followedUID, userID)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "insert into user fans err")
	}

	// 添加关注数
	err = srv.userStatRepo.IncrFollowCount(ctx, tx, userID, 1)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update user follow count err")
	}

	// 添加粉丝数
	err = srv.userStatRepo.IncrFollowerCount(ctx, tx, followedUID, 1)
	if err != nil {
		return errors.Wrap(err, "update user fans count err")
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "tx commit err")
	}

	return nil
}

// CancelUserFollow 取消用户关注
func (srv *userService) CancelUserFollow(ctx context.Context, userID uint64, followedUID uint64) error {
	db := model.GetDB()
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除关注
	err := srv.userFollowRepo.UpdateUserFollowStatus(ctx, tx, userID, followedUID, FollowStatusDelete)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update user follow err")
	}

	// 删除粉丝
	err = srv.userFollowRepo.UpdateUserFansStatus(ctx, tx, followedUID, userID, FollowStatusDelete)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update user follow err")
	}

	// 减少关注数
	err = srv.userStatRepo.IncrFollowCount(ctx, tx, userID, -1)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update user follow count err")
	}

	// 减少粉丝数
	err = srv.userStatRepo.IncrFollowerCount(ctx, tx, followedUID, -1)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update user fans count err")
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "tx commit err")
	}

	return nil
}

// GetFollowingUserList 获取正在关注的用户列表
func (srv *userService) GetFollowingUserList(ctx context.Context, userID uint64, lastID uint64, limit int) ([]*model.UserFollowModel, error) {
	if lastID == 0 {
		lastID = MaxID
	}
	userFollowList, err := srv.userFollowRepo.GetFollowingUserList(ctx, userID, lastID, limit)
	if err != nil {
		return nil, err
	}

	return userFollowList, nil
}

// GetFollowerUserList 获取粉丝用户列表
func (srv *userService) GetFollowerUserList(ctx context.Context, userID uint64, lastID uint64, limit int) ([]*model.UserFansModel, error) {
	if lastID == 0 {
		lastID = MaxID
	}
	userFollowerList, err := srv.userFollowRepo.GetFollowerUserList(ctx, userID, lastID, limit)
	if err != nil {
		return nil, err
	}

	return userFollowerList, nil
}

// Close close all user repo
func (srv *userService) Close() {
	srv.userRepo.Close()
	srv.userFollowRepo.Close()
	srv.userStatRepo.Close()
}
