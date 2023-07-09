package immich

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"sync"
	"time"
)

type MultipartWriter interface {
	WriteMultiPart(w *multipart.Writer) error
}

var SupportedMime = map[string]any{
	// IMAGES
	"image/heif":        nil,
	"image/heic":        nil,
	"image/jpeg":        nil,
	"image/png":         nil,
	"image/jpg":         nil,
	"image/gif":         nil,
	"image/dng":         nil,
	"image/x-adobe-dng": nil,
	"image/webp":        nil,
	"image/tiff":        nil,
	"image/nef":         nil,
	"image/x-nikon-nef": nil,

	// VIDEO
	"video/mp4":       nil,
	"video/webm":      nil,
	"video/quicktime": nil,
	"video/x-msvideo": nil,
	"video/3gpp":      nil,
}

func IsMimeSupported(mime string) (string, error) {
	_, ok := SupportedMime[mime]
	if !ok {
		return "", UnsupportedMedia(fmt.Errorf("%s is not supported", mime))
	}
	s := strings.SplitN(mime, "/", 1)
	return s[0], nil
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
