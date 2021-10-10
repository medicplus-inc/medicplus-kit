package structure

type ResolveStructure struct {
	Data     interface{}            `json:"data"`
	Metadata map[string]interface{} `json:"metadata"`
}

type RejectStructure struct {
	Code    int    `json:"-"`
	Error   error  `json:"error"`
	Message string `json:"message"`
}
