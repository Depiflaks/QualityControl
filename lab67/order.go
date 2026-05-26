package order

import (
	"errors"
	"fmt"
)

type OrderStatus string

const (
	StatusDraft     OrderStatus = "draft"
	StatusConfirmed OrderStatus = "confirmed"
	StatusCancelled OrderStatus = "cancelled"
)

var (
	ErrInvalidProductID   = errors.New("invalid product id")
	ErrInvalidQuantity    = errors.New("invalid quantity")
	ErrInvalidDiscount    = errors.New("invalid discount")
	ErrProductNotFound    = errors.New("product not found")
	ErrProductUnavailable = errors.New("product unavailable")
	ErrOrderIsEmpty       = errors.New("order is empty")
	ErrOrderIsNotDraft    = errors.New("order is not draft")
	ErrOrderIsCancelled   = errors.New("order is cancelled")
	ErrOrderIsConfirmed   = errors.New("order is confirmed")
)

type InventoryService interface {
	CheckAvailability(productID string, quantity int) (bool, error)
	GetPrice(productID string) (float64, error)
}

type OrderItem struct {
	ProductID string
	Quantity  int
	UnitPrice float64
}

type Order struct {
	Items           map[string]OrderItem
	Status          OrderStatus
	DiscountPercent float64

	inventory InventoryService
}

func NewOrder(inventory InventoryService) *Order {
	return &Order{
		Items:           make(map[string]OrderItem),
		Status:          StatusDraft,
		DiscountPercent: 0,
		inventory:       inventory,
	}
}

func (o *Order) AddItem(productID string, quantity int) error {
	if o.Status != StatusDraft {
		return ErrOrderIsNotDraft
	}

	if productID == "" {
		return ErrInvalidProductID
	}

	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	totalQuantity := quantity

	if item, exists := o.Items[productID]; exists {
		totalQuantity += item.Quantity
	}

	available, err := o.inventory.CheckAvailability(productID, totalQuantity)
	if err != nil {
		return err
	}

	if !available {
		return ErrProductUnavailable
	}

	price, err := o.inventory.GetPrice(productID)
	if err != nil {
		return err
	}

	o.Items[productID] = OrderItem{
		ProductID: productID,
		Quantity:  totalQuantity,
		UnitPrice: price,
	}

	return nil
}

func (o *Order) RemoveItem(productID string) error {
	if o.Status != StatusDraft {
		return ErrOrderIsNotDraft
	}

	if productID == "" {
		return ErrInvalidProductID
	}

	if _, exists := o.Items[productID]; !exists {
		return ErrProductNotFound
	}

	delete(o.Items, productID)

	return nil
}

func (o *Order) ApplyDiscount(percent float64) error {
	if o.Status != StatusDraft {
		return ErrOrderIsNotDraft
	}

	if percent < 0 || percent > 100 {
		return ErrInvalidDiscount
	}

	o.DiscountPercent = percent

	return nil
}

func (o *Order) CalculateTotal() float64 {
	total := 0.0

	for _, item := range o.Items {
		total += float64(item.Quantity) * item.UnitPrice
	}

	discount := total * o.DiscountPercent / 100

	return total - discount
}

func (o *Order) ConfirmOrder() error {
	if o.Status == StatusCancelled {
		return ErrOrderIsCancelled
	}

	if o.Status == StatusConfirmed {
		return ErrOrderIsConfirmed
	}

	if len(o.Items) == 0 {
		return ErrOrderIsEmpty
	}

	for _, item := range o.Items {
		available, err := o.inventory.CheckAvailability(item.ProductID, item.Quantity)
		if err != nil {
			return err
		}

		if !available {
			return fmt.Errorf("%w: %s", ErrProductUnavailable, item.ProductID)
		}
	}

	o.Status = StatusConfirmed

	return nil
}

func (o *Order) CancelOrder() error {
	if o.Status == StatusCancelled {
		return ErrOrderIsCancelled
	}

	if o.Status == StatusConfirmed {
		return ErrOrderIsConfirmed
	}

	o.Status = StatusCancelled

	return nil
}
