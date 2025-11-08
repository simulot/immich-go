package immich

import "context"

type SignupDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusRemoving UserStatus = "removing"
	UserStatusDeleted  UserStatus = "deleted"
)

type UserAdminResponseDto struct {
	CreatedAt            string     `json:"createdAt"`
	DeletedAt            string     `json:"deletedAt,omitempty"`
	Email                string     `json:"email"`
	ID                   string     `json:"id"`
	IsAdmin              bool       `json:"isAdmin"`
	Name                 string     `json:"name"`
	OauthID              string     `json:"oauthId"`
	ProfileChangedAt     string     `json:"profileChangedAt"`
	ProfileImagePath     string     `json:"profileImagePath"`
	QuotaSizeInBytes     int64      `json:"quotaSizeInBytes,omitempty"`
	QuotaUsageInBytes    int64      `json:"quotaUsageInBytes,omitempty"`
	ShouldChangePassword bool       `json:"shouldChangePassword"`
	Status               UserStatus `json:"status"`
	StorageLabel         string     `json:"storageLabel,omitempty"`
	UpdatedAt            string     `json:"updatedAt"`
	// AvatarColor          UserAvatarColor `json:"avatarColor"`
	// License              *UserLicense    `json:"license,omitempty"`
}

func (ic *ImmichClient) SignUpAdmin(ctx context.Context, email, password, name string) (User, error) {
	var user User

	err := ic.newServerCall(ctx, EndPointSignUpAdmin).
		do(getRequest("/api/sign-up", setAcceptJSON()), responseJSON(&user))
	return user, err
}

type LoginCredentialDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponseDto struct {
	AccessToken          string `json:"accessToken"`
	IsAdmin              bool   `json:"isAdmin"`
	IsOnboarded          bool   `json:"isOnboarded"`
	Name                 string `json:"name"`
	ProfileImagePath     string `json:"profileImagePath"`
	ShouldChangePassword bool   `json:"shouldChangePassword"`
	UserEmail            string `json:"userEmail"`
	UserID               string `json:"userId"`
}

func (ic *ImmichClient) UserLogin(ctx context.Context, email, password string) (LoginResponseDto, error) {
	user := LoginCredentialDto{
		Email:    email,
		Password: password,
	}

	var loginResponse LoginResponseDto

	err := ic.newServerCall(ctx, EndPointLogin).
		do(postRequest("/api/login", "application/json", setJSONBody(&user)), responseJSON(&loginResponse))
	return loginResponse, err
}

type AdminOnboardingUpdateDto struct {
	IsOnboarded bool `json:"isOnboarded"`
}

func (ic *ImmichClient) UpdateAdminOnboarding(ctx context.Context, isOnBoarded bool) error {
	onboarding := AdminOnboardingUpdateDto{
		IsOnboarded: isOnBoarded,
	}
	err := ic.newServerCall(ctx, EndPointUpdateAdminOnboarding).
		do(postRequest("/api/admin/onboarding", "application/json", setJSONBody(&onboarding)))
	return err
}
