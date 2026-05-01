package model

type Article struct {
	ID              string    `json:"id"`
	IDReadable      string    `json:"idReadable"`
	Summary         string    `json:"summary"`
	Content         string    `json:"content"`
	WikifiedContent string    `json:"wikifiedContent"`
	Project         *Project  `json:"project"`
	Reporter        *User     `json:"reporter"`
	Created         int64     `json:"created"`
	Updated         int64     `json:"updated"`
	Visibility      *Visibility `json:"visibility,omitempty"`
	Ordinal         int       `json:"ordinal"`
	ParentArticle   *Article  `json:"parentArticle,omitempty"`
	Tags            []Tag     `json:"tags"`
	CommentsCount   int       `json:"commentsCount"`
	Comments        []Comment `json:"comments"`
}

type Visibility struct {
	ID   string `json:"id"`
	Type string `json:"$type"`
}