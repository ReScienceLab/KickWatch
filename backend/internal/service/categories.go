package service

import "github.com/kickwatch/backend/internal/model"

// kickstarterCategories lists root and high-value subcategories.
// Root IDs and subcategory IDs confirmed from Kickstarter REST API public datasets.
var kickstarterCategories = []model.Category{
	// Root categories
	{ID: "1", Name: "Art", NameZh: "艺术"},
	{ID: "3", Name: "Comics", NameZh: "漫画"},
	{ID: "4", Name: "Crafts", NameZh: "手工艺"},
	{ID: "5", Name: "Dance", NameZh: "舞蹈"},
	{ID: "7", Name: "Design", NameZh: "设计"},
	{ID: "9", Name: "Fashion", NameZh: "时尚"},
	{ID: "10", Name: "Food", NameZh: "美食"},
	{ID: "11", Name: "Film & Video", NameZh: "影视"},
	{ID: "12", Name: "Games", NameZh: "游戏"},
	{ID: "13", Name: "Music", NameZh: "音乐"},
	{ID: "14", Name: "Photography", NameZh: "摄影"},
	{ID: "16", Name: "Technology", NameZh: "科技"},
	{ID: "17", Name: "Theater", NameZh: "戏剧"},
	{ID: "18", Name: "Publishing", NameZh: "出版"},

	// Design subcategories
	{ID: "28", Name: "Product Design", NameZh: "产品设计", ParentID: "7"},

	// Fashion subcategories
	{ID: "263", Name: "Apparel", NameZh: "服装", ParentID: "9"},

	// Film & Video subcategories
	{ID: "29", Name: "Animation", NameZh: "动画", ParentID: "11"},
	{ID: "303", Name: "Television", NameZh: "电视", ParentID: "11"},

	// Games subcategories
	{ID: "34", Name: "Tabletop Games", NameZh: "桌游", ParentID: "12"},
	{ID: "35", Name: "Video Games", NameZh: "电子游戏", ParentID: "12"},
	{ID: "270", Name: "Gaming Hardware", NameZh: "游戏硬件", ParentID: "12"},

	// Technology subcategories
	{ID: "52", Name: "Hardware", NameZh: "硬件", ParentID: "16"},
	{ID: "331", Name: "3D Printing", NameZh: "3D打印", ParentID: "16"},
	{ID: "337", Name: "Gadgets", NameZh: "数码产品", ParentID: "16"},
	{ID: "339", Name: "Sound", NameZh: "音频设备", ParentID: "16"},

	// Publishing subcategories
	{ID: "47", Name: "Fiction", NameZh: "小说", ParentID: "18"},
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
