package model

import "errors"

var (
	// ErrEmptyName は名前に空文字が指定された場合のエラー。
	ErrEmptyName = errors.New("name must not be empty")
	// ErrNotFound は対象が見つからない場合のエラー。
	ErrNotFound = errors.New("not found")
	// ErrInvalidIndex はインデックスが範囲外の場合のエラー。
	ErrInvalidIndex = errors.New("index out of range")
)
