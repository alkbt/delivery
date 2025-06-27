package http

import (
	"net/http"

	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/application/usecases/queries"
	"delivery/internal/core/domain/model/kernel"
	"delivery/internal/generated/servers"

	"github.com/labstack/echo/v4"
)

// Server implements the ServerInterface for handling HTTP requests.
// It coordinates between HTTP handlers and application use cases.
type Server struct {
	// Command handlers
	createCourierHandler commands.CreateCourierCommandHandler
	createOrderHandler   commands.CreateOrderCommandHandler

	// Query handlers
	getAllCouriersHandler       queries.GetAllCouriersQueryHandler
	getUncompletedOrdersHandler queries.GetUncompletedOrdersQueryHandler
}

// NewServer creates a new HTTP server with the required command and query handlers.
func NewServer(
	createCourierHandler commands.CreateCourierCommandHandler,
	createOrderHandler commands.CreateOrderCommandHandler,
	getAllCouriersHandler queries.GetAllCouriersQueryHandler,
	getUncompletedOrdersHandler queries.GetUncompletedOrdersQueryHandler,
) *Server {
	return &Server{
		createCourierHandler:        createCourierHandler,
		createOrderHandler:          createOrderHandler,
		getAllCouriersHandler:       getAllCouriersHandler,
		getUncompletedOrdersHandler: getUncompletedOrdersHandler,
	}
}

// GetCouriers handles GET /api/v1/couriers - retrieves all couriers.
func (s *Server) GetCouriers(ctx echo.Context) error {
	query := queries.NewGetAllCouriersQuery()

	couriers, err := s.getAllCouriersHandler.Handle(ctx.Request().Context(), query)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, servers.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to retrieve couriers",
		})
	}

	response := make([]servers.Courier, len(couriers))
	for i, courier := range couriers {
		googleUUID := courier.ID.Bytes()

		response[i] = servers.Courier{
			Id: googleUUID,
			Location: servers.Location{
				X: int(courier.Location.X()),
				Y: int(courier.Location.Y()),
			},
			Name: courier.Name,
		}
	}

	return ctx.JSON(http.StatusOK, response)
}

// CreateCourier handles POST /api/v1/couriers - creates a new courier.
func (s *Server) CreateCourier(ctx echo.Context) error {
	var newCourier servers.NewCourier
	if err := ctx.Bind(&newCourier); err != nil {
		return ctx.JSON(http.StatusBadRequest, servers.Error{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
		})
	}

	// Generate random location for the courier
	location, err := kernel.NewRandomLocation()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, servers.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate courier location",
		})
	}

	cmd, err := commands.NewCreateCourierCommand(newCourier.Name, newCourier.Speed, location)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, servers.Error{
			Code:    http.StatusBadRequest,
			Message: "Invalid courier data: " + err.Error(),
		})
	}

	if handleErr := s.createCourierHandler.Handle(ctx.Request().Context(), cmd); handleErr != nil {
		return ctx.JSON(http.StatusConflict, servers.Error{
			Code:    http.StatusConflict,
			Message: "Failed to create courier",
		})
	}

	return ctx.NoContent(http.StatusCreated)
}

// CreateOrder handles POST /api/v1/orders - creates a new order.
func (s *Server) CreateOrder(ctx echo.Context) error {
	// For this API, we'll create an order with a random location and volume
	// Since the OpenAPI spec doesn't specify request body for order creation
	orderID := kernel.NewUUID()
	street := "Auto-generated order"
	volume := 10 // Default volume

	cmd, err := commands.NewCreateOrderCommand(orderID, street, volume)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, servers.Error{
			Code:    http.StatusBadRequest,
			Message: "Invalid order data: " + err.Error(),
		})
	}

	if handleErr := s.createOrderHandler.Handle(ctx.Request().Context(), cmd); handleErr != nil {
		return ctx.JSON(http.StatusInternalServerError, servers.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create order",
		})
	}

	return ctx.NoContent(http.StatusCreated)
}

// GetOrders handles GET /api/v1/orders/active - retrieves all uncompleted orders.
func (s *Server) GetOrders(ctx echo.Context) error {
	query := queries.NewGetUncompletedOrdersQuery()

	orders, err := s.getUncompletedOrdersHandler.Handle(ctx.Request().Context(), query)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, servers.Error{
			Code:    http.StatusInternalServerError,
			Message: "Failed to retrieve orders",
		})
	}

	response := make([]servers.Order, len(orders))
	for i, order := range orders {
		googleUUID := order.ID.Bytes()

		response[i] = servers.Order{
			Id: googleUUID,
			Location: servers.Location{
				X: int(order.Location.X()),
				Y: int(order.Location.Y()),
			},
		}
	}

	return ctx.JSON(http.StatusOK, response)
}
