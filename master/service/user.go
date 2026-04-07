package service

import (
	"errors"
	"goworkers/config"
	"goworkers/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func Register(username, password, email string) (*model.User, error) {
	var repeat model.User
	// search db to avoid conflict
	result_username := config.DB.Where("username = ?", username).First(&repeat)
	if result_username.Error == nil {
		return nil, errors.New("username exists")
	}

	result_email := config.DB.Where("email = ?", email).First(&repeat)
	if result_email.Error == nil {
		return nil, errors.New("email exists")
	}
	// email password format verified

	// password encrypto
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to encrypt passowrd")
	}
	//register
	var user = model.User{
		Username: username,
		Password: string(hash),
		Email:    email,
	}
	if result := config.DB.Create(&user); result.Error != nil {
		return nil, errors.New("failed to create user")
	}

	return &user, nil
}

func Login(identifier, password string) (*model.User, error) {
	var user model.User
	//verified the user exist or not
	if result := config.DB.Where("username = ? OR email = ?", identifier, identifier).First(&user); result.Error != nil {
		return nil, errors.New("user not found, check your email or username")
	}

	//validate the password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("the wrong password")
	}

	return &user, nil
}

func GetProfile(userID uint) (*model.User, error) {
	var user model.User
	if result := config.DB.First(&user, userID); result.Error != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func UpdateProfile(userID uint, username, email string) (*model.User, error) {
	var user model.User
	if result := config.DB.First(&user, userID); result.Error != nil {
		return nil, errors.New("user not found")
	}

	if username != "" && username != user.Username {
		var exist model.User
		if config.DB.Where("username = ?", username).First(&exist).Error == nil {
			return nil, errors.New("username exists")
		}
		user.Username = username
	}

	if email != "" && email != user.Email {
		var exist model.User
		if config.DB.Where("email = ?", email).First(&exist).Error == nil {
			return nil, errors.New("email exists")
		}
		user.Email = email
	}

	if result := config.DB.Save(&user); result.Error != nil {
		return nil, errors.New("failed to update profile")
	}
	return &user, nil
}

func GenerateToken(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte("spriple-jwt-key"))
}
