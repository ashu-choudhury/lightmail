package models

type User struct {
	ID       int    `xorm:"id unsigned int not null pk autoincr"`
	Account  string `xorm:"varchar(190) notnull unique comment('Account login name local@domain')"`
	Name     string `xorm:"varchar(50) notnull comment('Display Name')"`
	Password string `xorm:"char(32) notnull comment('Password hash')"`
	Disabled int    `xorm:"disabled unsigned int not null default(0) comment('0: enabled, 1: disabled')"`
	IsAdmin  int    `xorm:"is_admin unsigned int not null default(0) comment('0: regular user, 1: admin')"`
	Gender   string `xorm:"varchar(10) default('') comment('Gender: male, female, other')"`
}

func (p User) TableName() string {
	return "user"
}
