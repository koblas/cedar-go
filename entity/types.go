package entity

type Entity struct {
	Uid   string `json:"uid"`
	Attrs map[string]any
	// Parents []EntityValue
}

type EntityMap struct {
	Entities []Entity
}

func ParseJson(input []byte) {

}
