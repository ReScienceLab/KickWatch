package service

import "github.com/kickwatch/backend/internal/model"

// kickstarterCategories lists root and high-value subcategories.
// Root IDs and subcategory IDs confirmed from Kickstarter REST API public datasets.
var kickstarterCategories = []model.Category{
	// Root categories
	{ID: "1", Name: "Art"},
	{ID: "3", Name: "Comics"},
	{ID: "4", Name: "Crafts"},
	{ID: "5", Name: "Dance"},
	{ID: "7", Name: "Design"},
	{ID: "9", Name: "Fashion"},
	{ID: "10", Name: "Food"},
	{ID: "11", Name: "Film & Video"},
	{ID: "12", Name: "Games"},
	{ID: "13", Name: "Music"},
	{ID: "14", Name: "Photography"},
	{ID: "16", Name: "Technology"},
	{ID: "17", Name: "Theater"},
	{ID: "18", Name: "Publishing"},

	// Design subcategories
	{ID: "28", Name: "Product Design", ParentID: "7"},

	// Fashion subcategories
	{ID: "263", Name: "Apparel", ParentID: "9"},

	// Film & Video subcategories
	{ID: "29", Name: "Animation", ParentID: "11"},
	{ID: "303", Name: "Television", ParentID: "11"},

	// Games subcategories
	{ID: "34", Name: "Tabletop Games", ParentID: "12"},
	{ID: "35", Name: "Video Games", ParentID: "12"},
	{ID: "270", Name: "Gaming Hardware", ParentID: "12"},

	// Technology subcategories
	{ID: "52", Name: "Hardware", ParentID: "16"},
	{ID: "331", Name: "3D Printing", ParentID: "16"},
	{ID: "337", Name: "Gadgets", ParentID: "16"},
	{ID: "339", Name: "Sound", ParentID: "16"},

	// Publishing subcategories
	{ID: "47", Name: "Fiction", ParentID: "18"},
}

// crawlCategories defines all category IDs to crawl and their page depth.
// Root categories get deeper crawls; subcategories are more focused.
var crawlCategories = []crawlCategory{
	// Root categories — 10 pages each (~200 items)
	{"1", 10}, {"3", 10}, {"4", 10}, {"5", 10},
	{"7", 10}, {"9", 10}, {"10", 10}, {"11", 10},
	{"12", 10}, {"13", 10}, {"14", 10}, {"16", 10},
	{"17", 10}, {"18", 10},

	// High-value subcategories — 5 pages each (~100 items)
	{"28", 5},  // Product Design
	{"34", 5},  // Tabletop Games
	{"35", 5},  // Video Games
	{"270", 5}, // Gaming Hardware
	{"52", 5},  // Hardware
	{"331", 5}, // 3D Printing
	{"337", 5}, // Gadgets
	{"339", 5}, // Sound
	{"47", 5},  // Fiction
	{"29", 5},  // Animation
	{"303", 5}, // Television
	{"263", 5}, // Apparel
}

type crawlCategory struct {
	ID        string
	PageDepth int
}
