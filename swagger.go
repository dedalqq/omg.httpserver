package httpserver

const (
	TypeObject  = "object"
	TypeString  = "string"
	TypeInteger = "integer"
	TypeBool    = "boolean"
)

type apiType struct {
	Type        string              `json:"type,omitempty"`
	Description string              `json:"description,omitempty"`
	Required    bool                `json:"required,omitempty"`
	Format      string              `json:"format,omitempty"`
	Properties  OrderedMap[apiType] `json:"properties,omitempty"`
}

type apiInfo struct {
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Title       string `json:"title,omitempty"`
}

type apiTags struct {
}

type apiSecurity struct {
}

type apiSchema struct {
	Ref string `json:"$ref"`
}

type apiParameter struct {
	In          string     `json:"in,omitempty"`
	Name        string     `json:"name,omitempty"`
	Type        string     `json:"type,omitempty"`
	Description string     `json:"description,omitempty"`
	Required    bool       `json:"required,omitempty"`
	Schema      *apiSchema `json:"schema,omitempty"`
}

type apiEndpoint struct {
	Get    *apiHandler `json:"get,omitempty"`
	Post   *apiHandler `json:"post,omitempty"`
	Put    *apiHandler `json:"put,omitempty"`
	Delete *apiHandler `json:"delete,omitempty"`
	Patch  *apiHandler `json:"patch,omitempty"`
}

type apiHandler struct {
	Tags        []string                 `json:"tags,omitempty"`
	Summary     string                   `json:"summary,omitempty"`
	Description string                   `json:"description,omitempty"`
	OperationId string                   `json:"operationId,omitempty"`
	Consumes    []string                 `json:"consumes,omitempty"`
	Produces    []string                 `json:"produces,omitempty"`
	Parameters  []apiParameter           `json:"parameters,omitempty"`
	Responses   OrderedMap[apiParameter] `json:"responses,omitempty"`
	Security    []apiSecurity            `json:"security,omitempty"`
}

type Swagger struct {
	Swagger     string                  `json:"swagger"`
	Info        apiInfo                 `json:"info"`
	Host        string                  `json:"host,omitempty"`
	BasePath    string                  `json:"basePath,omitempty"`
	Tags        []*apiTags              `json:"tags,omitempty"`
	Schemes     []string                `json:"schemes,omitempty"`
	Paths       OrderedMap[apiEndpoint] `json:"paths,omitempty"`
	Definitions OrderedMap[apiType]     `json:"definitions"`
}
