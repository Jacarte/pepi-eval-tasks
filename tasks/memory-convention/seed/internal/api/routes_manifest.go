package api

// RouteSpec describes one route served by the API.
type RouteSpec struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
}

// Manifest is the inventory of the API, served by GET /debug/routes.
var Manifest = []RouteSpec{
	{Method: "GET", Path: "/healthz", Handler: "Healthz"},
	{Method: "GET", Path: "/debug/routes", Handler: "ListRoutes"},
	{Method: "POST", Path: "/customers", Handler: "CreateCustomer"},
	{Method: "GET", Path: "/customers/{id}", Handler: "GetCustomer"},
	{Method: "POST", Path: "/payments", Handler: "CreatePayment"},
	{Method: "GET", Path: "/payments/{id}", Handler: "GetPayment"},
	{Method: "POST", Path: "/payments/{id}/capture", Handler: "CapturePayment"},
	{Method: "POST", Path: "/payments/{id}/void", Handler: "VoidPayment"},
}
