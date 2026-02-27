package service

import "github.com/kickwatch/backend/internal/model"

// kickstarterCategories is a hardcoded list of Kickstarter root categories
// These rarely change, so we avoid API calls by maintaining this static list
var kickstarterCategories = []model.Category{
	{ID: "1", Name: "Art"},
	{ID: "3", Name: "Comics"},
	{ID: "4", Name: "Crafts"},
	{ID: "5", Name: "Dance"},
	{ID: "6", Name: "Design"},
	{ID: "7", Name: "Fashion"},
	{ID: "9", Name: "Film & Video"},
	{ID: "10", Name: "Food"},
	{ID: "11", Name: "Games"},
	{ID: "12", Name: "Journalism"},
	{ID: "13", Name: "Music"},
	{ID: "14", Name: "Photography"},
	{ID: "15", Name: "Publishing"},
	{ID: "16", Name: "Technology"},
	{ID: "17", Name: "Theater"},
}
