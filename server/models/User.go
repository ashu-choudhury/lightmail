package models

type User struct {
	ID       int    `xorm:"id unsigned int not null pk autoincr" json:"id"`
	Account  string `xorm:"varchar(190) notnull unique comment('Account login name local@domain')" json:"account"`
	Name     string `xorm:"varchar(50) notnull comment('Display Name')" json:"name"`
	Password string `xorm:"char(32) notnull comment('Password hash')" json:"-"`
	Disabled int    `xorm:"disabled unsigned int not null default(0) comment('0: enabled, 1: disabled')" json:"disabled"`
	IsAdmin  int    `xorm:"is_admin unsigned int not null default(0) comment('0: regular user, 1: admin')" json:"is_admin"`
	Gender   string `xorm:"varchar(10) default('') comment('Gender: male, female, other')" json:"gender"`
}

func (p User) TableName() string {
	return "user"
}
