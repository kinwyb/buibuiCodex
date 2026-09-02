package db

import "gorm.io/gorm"

type IStorage interface {
	GetDB() *gorm.DB
	Close() error
}

type Data struct {
	storage IStorage
	session ISession
	cron    ITask
}

func NewData(storage IStorage) *Data {
	ret := &Data{
		storage: storage,
	}
	if storage != nil && storage.GetDB() != nil {
		db := storage.GetDB()
		ret.session = newSessionStorage(db)
		ret.cron = newTaskStorage(db)
		ret.session.Init()
		ret.cron.Init()
	}
	return ret
}

func (d *Data) Session() ISession {
	if d.storage == nil {
		d.session = &sessionDefautStorage{}
	}
	return d.session
}

func (d *Data) Cron() ITask {
	if d.cron == nil {
		d.cron = &taskDefaultStorage{}
	}
	return d.cron
}

func (d *Data) Close() {
	if d.storage != nil {
		d.storage.Close()
	}
}
