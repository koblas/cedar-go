package schema

// type JsonEntityValue struct {
// 	Type string `json:"type,omitempty"`
// 	Id   string `json:"id,omitempty"`
// 	// This is the second form
// 	Entity *struct {
// 		Type string `json:"type,omitempty"`
// 		Id   string `json:"id,omitempty"`
// 	} `json:"__entity,omitempty"`
// }

type JsonEntityValue map[string]any

type JsonEntityItem struct {
	Uid     JsonEntityValue   `json:"uid"`
	Parents []JsonEntityValue `json:"parents"`
	Attrs   map[string]any    `json:"attrs"`
}

type JsonEntities []JsonEntityItem
