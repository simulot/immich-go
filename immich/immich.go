package immich

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type UnsupportedMedia struct {
	msg string
}

func (u UnsupportedMedia) Error() string {
	return u.msg
}

func (u UnsupportedMedia) Is(target error) bool {
	_, ok := target.(*UnsupportedMedia)
	return ok
}

type PingResponse struct {
	Res string `json:"res"`
}

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	FirstName            string    `json:"firstName"`
	LastName             string    `json:"lastName"`
	StorageLabel         string    `json:"storageLabel"`
	ExternalPath         string    `json:"externalPath"`
	ProfileImagePath     string    `json:"profileImagePath"`
	ShouldChangePassword bool      `json:"shouldChangePassword"`
	IsAdmin              bool      `json:"isAdmin"`
	CreatedAt            time.Time `json:"createdAt"`
	DeletedAt            time.Time `json:"deletedAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	OauthID              string    `json:"oauthId"`
}

type List[T comparable] struct {
	list []T
	lock sync.RWMutex
}

func (l *List[T]) Includes(v T) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	for i := range l.list {
		if l.list[i] == v {
			return true
		}
	}
	return false
}

func (l *List[T]) Push(v T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list = append(l.list, v)
}

func (l *List[T]) MarshalJSON() ([]byte, error) {
	return nil, errors.New("MarshalJSON not implemented for List[T]")
}

func (l *List[T]) UnmarshalJSON(data []byte) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.list == nil {
		l.list = []T{}
	}
	return json.Unmarshal(data, &l.list)
}

type StringList struct{ List[string] }

type myBool bool

func (b myBool) String() string {
	if b {
		return "true"
	}
	return "false"
}
