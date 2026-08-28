package db

import "gorm.io/gorm"

type IStorage interface {
	GetDB() *gorm.DB
	Close() error
}

type Data struct {
	storage IStorage
	session ISessionStorage
}

func NewData(storage IStorage) *Data {
	ret := &Data{
		storage: storage,
	}
	if storage != nil && storage.GetDB() != nil {
		db := storage.GetDB()
		ret.session = NewSessionStorage(db)
		ret.session.Init()
	}
	return ret
}

func (d *Data) Session() ISessionStorage {
	if d.storage == nil {
		d.session = &sessionDefautStorage{}
	}
	return d.session
}

func (d *Data) Close() {
	if d.storage != nil {
		d.storage.Close()
	}
}
