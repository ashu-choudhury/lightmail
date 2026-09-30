package models

import "encoding/json"

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

// UnmarshalJSON provides seamless backward and forward compatibility
// across JSON with lowercase (id, is_admin) and PascalCase (ID, IsAdmin) properties.
func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User
	aux := &struct {
		ID             *int    `json:"id"`
		Account        *string `json:"account"`
		Name           *string `json:"name"`
		Disabled       *int    `json:"disabled"`
		IsAdmin        *int    `json:"is_admin"`
		Gender         *string `json:"gender"`
		LegacyID       *int    `json:"ID"`
		LegacyAccount  *string `json:"Account"`
		LegacyName     *string `json:"Name"`
		LegacyDisabled *int    `json:"Disabled"`
		LegacyIsAdmin  *int    `json:"IsAdmin"`
		LegacyGender   *string `json:"Gender"`
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ID != nil {
		u.ID = *aux.ID
	} else if aux.LegacyID != nil {
		u.ID = *aux.LegacyID
	}
	if aux.Account != nil {
		u.Account = *aux.Account
	} else if aux.LegacyAccount != nil {
		u.Account = *aux.LegacyAccount
	}
	if aux.Name != nil {
		u.Name = *aux.Name
	} else if aux.LegacyName != nil {
		u.Name = *aux.LegacyName
	}
	if aux.Disabled != nil {
		u.Disabled = *aux.Disabled
	} else if aux.LegacyDisabled != nil {
		u.Disabled = *aux.LegacyDisabled
	}
	if aux.IsAdmin != nil {
		u.IsAdmin = *aux.IsAdmin
	} else if aux.LegacyIsAdmin != nil {
		u.IsAdmin = *aux.LegacyIsAdmin
	}
	if aux.Gender != nil {
		u.Gender = *aux.Gender
	} else if aux.LegacyGender != nil {
		u.Gender = *aux.LegacyGender
	}
	return nil
}
