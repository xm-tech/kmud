package database

import (
	"kmud/utils"
)

const (
	userColorMode string = "colormode"
)

type User struct {
	DbObject  `bson:",inline"`
	colorMode utils.ColorMode
	online    bool
}

func NewUser(name string) User {
	var user User
	user.initDbObject(userType)
	commitObject(session, getCollection(session, cUsers), user)

	user.SetName(name)
	user.SetColorMode(utils.ColorModeNone)

	user.SetOnline(false)

	return user
}

func (self *User) SetOnline(online bool) {
	self.online = online
}

func (self *User) Online() bool {
	return self.online
}

func (self *User) SetColorMode(cm utils.ColorMode) {
	self.setField(userColorMode, cm)
}

func (self *User) GetColorMode() utils.ColorMode {
	return self.getField(userColorMode).(utils.ColorMode)
}

// vim: nocindent
