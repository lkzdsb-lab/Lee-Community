package pkg

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired      = errors.New("token expired")
	ErrTokenInvalid      = errors.New("token invalid")
	ErrRefreshExpired    = errors.New("refresh expired")
	ErrRefreshInvalid    = errors.New("refresh invalid")
	ErrTokenParseFailure = errors.New("token parse failure")
)

const (
	AccessTTL  = time.Minute * 30
	RefreshTTL = time.Hour * 24
)

// AccessSecret 先写死，后面放 config
var AccessSecret = []byte("secret-key")
var RefreshSecret = []byte("refresh-key")

type Claims struct {
	UserID uint64 `json:"user_id"`
	Role   int    `json:"role"` // 后续需要完善
	jwt.RegisteredClaims
}

type Pair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GeneratePair(userID uint64) (*Pair, error) {
	now := time.Now()

	access := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
			Subject:   "access",
		},
	})
	accessToken, err := access.SignedString(AccessSecret)
	if err != nil {
		return nil, err
	}

	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTTL)),
			Subject:   "refresh",
			// 可加 ID 作为 jti：ID: uuid.NewString(),
		},
	})
	refreshToken, err := refresh.SignedString(RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &Pair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// ParseAccess 解析 access
func ParseAccess(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return AccessSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrTokenInvalid
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrTokenExpired
		default:
			return nil, err
		}
	}
	if !token.Valid {
		return nil, ErrTokenParseFailure
	}
	return token.Claims.(*Claims), nil
}

// Refresh 刷新接口
func Refresh(refreshToken string) (*Pair, error) {
	// 解析 refresh
	token, err := jwt.ParseWithClaims(refreshToken, &Claims{}, func(t *jwt.Token) (any, error) {
		return RefreshSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrRefreshInvalid
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrRefreshExpired
		}
		return nil, err
	}
	if !token.Valid {
		return nil, err
	}
	claims := token.Claims.(*Claims)
	// 可在此检查 jti 是否已吊销/过期、是否匹配设备等
	return GeneratePair(claims.UserID)
}
