package telegram

import "time"

type ChannelInfo struct {
	Username    string `json:"username"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Subscribers string `json:"subscribers,omitempty"`
	URL         string `json:"url"`
}

type Post struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Text      string    `json:"text,omitempty"`
	Images    []string  `json:"images,omitempty"`
	Views     string    `json:"views,omitempty"`
	Datetime  time.Time `json:"datetime,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
}

type LatestPostsOptions struct {
	Limit        int
	BeforePostID int
	BeforeTime   time.Time
}

type SearchOptions struct {
	Query string
	Limit int
}
