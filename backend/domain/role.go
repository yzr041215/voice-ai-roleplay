package domain

type Role struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}
type RoleWithoutPrompt struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type RoleList struct {
	Roles []RoleWithoutPrompt `json:"roles"`
}
