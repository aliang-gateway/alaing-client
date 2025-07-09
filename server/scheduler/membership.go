package scheduler

import (
	"time"

	"nursor.org/nursorgate/common/model"
)

func AuthUser(token string) bool {
	// 如果用户名在登录状态中，并且登录状态未过期，则返回true; 如果登录状态过期，则删除登录状态
	if loginStatus[token] != nil {
		if loginStatus[token].LastLogin.Before(time.Now().Add(-5 * time.Minute)) {
			delete(loginStatus, token)
		} else {
			now := time.Now()
			loginStatus[token].LastLogin = now
			if loginStatus[token].ExpiredAt.Before(now) {
				delete(loginStatus, token)
				return false
			}
			print("user is valid in cache")
			return true
		}
	}
	print("user is not in cache")
	db := model.GetDB()

	user := model.User{}
	db.Unscoped().Where("access_token = ?", token).First(&user)
	if user.ID == 0 {
		return false
	}
	if user.ExpiredAt.Before(time.Now()) {
		return false
	}
	now := time.Now()
	loginStatus[token] = &model.LoginStatus{
		Username:  user.Name,
		ExpiredAt: user.ExpiredAt,
		LastLogin: now,
		UserID:    int(user.ID),
	}

	return true

}

func IsMembershipValid(id int) bool {
	user := model.User{}
	db := model.GetDB()
	db.Where("id = ?", id).First(&user)
	if user.ID == 0 {
		return false
	}
	if user.ExpiredAt.Before(time.Now()) {
		return false
	}
	now := time.Now()
	loginStatus[user.Name] = &model.LoginStatus{
		Username:  user.Name,
		ExpiredAt: user.ExpiredAt,
		LastLogin: now,
		UserID:    int(user.ID),
	}
	return true
}
