/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package model

import "gorm.io/gorm"

type Model interface {
	ModelAgent
}

type model struct {
	db *gorm.DB
}

func New(db *gorm.DB) Model {
	return &model{
		db: db,
	}
}
