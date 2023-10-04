package httpserver

type SwaggerInfo struct {
}

type SwaggerType struct {
}

type SwaggerResponse struct {
}

type SwaggerSecurity struct {
}

type SwaggerParameter struct {
}

type SwaggerEndpoint struct {
	Get *SwaggerHandler `json:"get"`
}

type SwaggerHandler struct {
	Tags        []string           `json:"tags"`
	Summary     string             `json:"summary"`
	Description string             `json:"description"`
	OperationId string             `json:"operationId"`
	Consumes    []string           `json:"consumes"`
	Produces    []string           `json:"produces"`
	Parameters  []SwaggerParameter `json:"parameters"`
	Responses   []SwaggerResponse  `json:"responses"`
	Security    []SwaggerSecurity  `json:"security"`
}

type SwaggerPath struct {
	Path     string
	Endpoint SwaggerEndpoint
}

type Swagger struct {
	Swagger  string         `json:"swagger"`
	Info     *SwaggerInfo   `json:"info"`
	Host     string         `json:"host"`
	BasePath string         `json:"basePath"`
	Tags     []*SwaggerType `json:"tags"`
	Schemes  []string       `json:"schemes"`
	Paths    []*SwaggerPath `json:"paths"`
}
