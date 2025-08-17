package rest

var _ StrictServerInterface = new(API)

type API struct {
	ActuatorsHandler
	UsersHandler
}
