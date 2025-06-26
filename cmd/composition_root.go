package cmd

import (
	"delivery/internal/adapters/out/postgres"
	"delivery/internal/core/application/usecases/commands"
	"delivery/internal/core/application/usecases/queries"

	"gorm.io/gorm"
)

type CompositionRoot struct {
	gormDB     *gorm.DB
	uowFactory postgres.GormUnitOfWorkFactory
}

func NewCompositionRoot(_ Config, gormDB *gorm.DB) CompositionRoot {
	return CompositionRoot{
		gormDB:     gormDB,
		uowFactory: *postgres.NewGormUnitOfWorkFactory(gormDB),
	}
}

func (c *CompositionRoot) CreateAddCourierStorageCommandHandler() commands.AddCourierStorageCommandHandler {
	var f commands.CourierUoWFactory = FuncCourierUoWFactory(func() commands.CourierUoW {
		return c.uowFactory.Create()
	})
	return commands.NewAddCourierStorageCommandHandler(f)
}

func (c *CompositionRoot) CreateCreateCourierCommandHandler() commands.CreateCourierCommandHandler {
	var f commands.CourierUoWFactory = FuncCourierUoWFactory(func() commands.CourierUoW {
		return c.uowFactory.Create()
	})
	return commands.NewCreateCourierCommandHandler(f)
}

func (c *CompositionRoot) CreateCreateOrderCommandHandler() commands.CreateOrderCommandHandler {
	var f commands.OrderUoWFactory = FuncOrderUoWFactory(func() commands.OrderUoW {
		return c.uowFactory.Create()
	})
	return commands.NewCreateOrderCommandHandler(f)
}

func (c *CompositionRoot) CreateMoveCouriersCommandHandler() commands.MoveCouriersCommandHandler {
	var f commands.UoWFactory = FuncUoWFactory(func() commands.UoW {
		return c.uowFactory.Create()
	})
	return commands.NewMoveCouriersCommandHandler(f)
}

func (c *CompositionRoot) CreateAssignCourierCommandHandler() commands.AssignCourierCommandHandler {
	var f commands.UoWFactory = FuncUoWFactory(func() commands.UoW {
		return c.uowFactory.Create()
	})
	return commands.NewAssignCourierCommandHandler(f)
}

func (c *CompositionRoot) CreateGetAllCouriersQueryHandler() queries.GetAllCouriersQueryHandler {
	return queries.NewGetAllCouriersQueryHandler(c.gormDB)
}

func (c *CompositionRoot) CreateGetUncompletedOrdersQueryHandler() queries.GetUncompletedOrdersQueryHandler {
	return queries.NewGetUncompletedOrdersQueryHandler(c.gormDB)
}

type FuncCourierUoWFactory func() commands.CourierUoW

func (f FuncCourierUoWFactory) Create() commands.CourierUoW {
	return f()
}

type FuncOrderUoWFactory func() commands.OrderUoW

func (f FuncOrderUoWFactory) Create() commands.OrderUoW {
	return f()
}

type FuncUoWFactory func() commands.UoW

func (f FuncUoWFactory) Create() commands.UoW {
	return f()
}
