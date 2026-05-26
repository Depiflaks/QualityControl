package order

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockInventoryService struct {
	mock.Mock
}

func (m *MockInventoryService) CheckAvailability(productID string, quantity int) (bool, error) {
	args := m.Called(productID, quantity)
	return args.Bool(0), args.Error(1)
}

func (m *MockInventoryService) GetPrice(productID string) (float64, error) {
	args := m.Called(productID)
	return args.Get(0).(float64), args.Error(1)
}

func TestNewOrderCreatesDraftOrder(t *testing.T) {
	inventory := new(MockInventoryService)

	order := NewOrder(inventory)

	require.NotNil(t, order)
	require.Empty(t, order.Items)
	require.Equal(t, StatusDraft, order.Status)
	require.Equal(t, 0.0, order.DiscountPercent)
}

func TestAddItemAddsNewItem(t *testing.T) {
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 2).Return(true, nil).Once()
	inventory.On("GetPrice", "product-1").Return(100.0, nil).Once()

	order := NewOrder(inventory)

	err := order.AddItem("product-1", 2)

	require.NoError(t, err)
	require.Len(t, order.Items, 1)
	require.Equal(t, OrderItem{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 100,
	}, order.Items["product-1"])

	inventory.AssertExpectations(t)
}

func TestAddItemIncreasesExistingItemQuantity(t *testing.T) {
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 2).Return(true, nil).Once()
	inventory.On("GetPrice", "product-1").Return(100.0, nil).Once()
	inventory.On("CheckAvailability", "product-1", 5).Return(true, nil).Once()
	inventory.On("GetPrice", "product-1").Return(90.0, nil).Once()

	order := NewOrder(inventory)

	require.NoError(t, order.AddItem("product-1", 2))
	err := order.AddItem("product-1", 3)

	require.NoError(t, err)
	require.Equal(t, 5, order.Items["product-1"].Quantity)
	require.Equal(t, 90.0, order.Items["product-1"].UnitPrice)

	inventory.AssertExpectations(t)
}

func TestAddItemReturnsErrorWhenOrderIsNotDraft(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusConfirmed

	err := order.AddItem("product-1", 1)

	require.ErrorIs(t, err, ErrOrderIsNotDraft)
	inventory.AssertNotCalled(t, "CheckAvailability")
	inventory.AssertNotCalled(t, "GetPrice")
}

func TestAddItemReturnsErrorWhenProductIDIsEmpty(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.AddItem("", 1)

	require.ErrorIs(t, err, ErrInvalidProductID)
	inventory.AssertNotCalled(t, "CheckAvailability")
	inventory.AssertNotCalled(t, "GetPrice")
}

func TestAddItemReturnsErrorWhenQuantityIsInvalid(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.AddItem("product-1", 0)

	require.ErrorIs(t, err, ErrInvalidQuantity)
	inventory.AssertNotCalled(t, "CheckAvailability")
	inventory.AssertNotCalled(t, "GetPrice")
}

func TestAddItemReturnsInventoryError(t *testing.T) {
	inventoryErr := errors.New("inventory error")
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 1).Return(false, inventoryErr).Once()

	order := NewOrder(inventory)

	err := order.AddItem("product-1", 1)

	require.ErrorIs(t, err, inventoryErr)
	require.Empty(t, order.Items)
	inventory.AssertExpectations(t)
	inventory.AssertNotCalled(t, "GetPrice")
}

func TestAddItemReturnsErrorWhenProductIsUnavailable(t *testing.T) {
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 10).Return(false, nil).Once()

	order := NewOrder(inventory)

	err := order.AddItem("product-1", 10)

	require.ErrorIs(t, err, ErrProductUnavailable)
	require.Empty(t, order.Items)
	inventory.AssertExpectations(t)
	inventory.AssertNotCalled(t, "GetPrice")
}

func TestAddItemReturnsErrorWhenPriceCannotBeLoaded(t *testing.T) {
	priceErr := errors.New("price error")
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 1).Return(true, nil).Once()
	inventory.On("GetPrice", "product-1").Return(0.0, priceErr).Once()

	order := NewOrder(inventory)

	err := order.AddItem("product-1", 1)

	require.ErrorIs(t, err, priceErr)
	require.Empty(t, order.Items)
	inventory.AssertExpectations(t)
}

func TestRemoveItemRemovesExistingItem(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Items["product-1"] = OrderItem{
		ProductID: "product-1",
		Quantity:  1,
		UnitPrice: 100,
	}

	err := order.RemoveItem("product-1")

	require.NoError(t, err)
	require.Empty(t, order.Items)
}

func TestRemoveItemReturnsErrorWhenOrderIsNotDraft(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusConfirmed

	err := order.RemoveItem("product-1")

	require.ErrorIs(t, err, ErrOrderIsNotDraft)
}

func TestRemoveItemReturnsErrorWhenProductIDIsEmpty(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.RemoveItem("")

	require.ErrorIs(t, err, ErrInvalidProductID)
}

func TestRemoveItemReturnsErrorWhenProductDoesNotExist(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.RemoveItem("missing-product")

	require.ErrorIs(t, err, ErrProductNotFound)
}

func TestApplyDiscountChangesDiscountPercent(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.ApplyDiscount(15)

	require.NoError(t, err)
	require.Equal(t, 15.0, order.DiscountPercent)
}

func TestApplyDiscountAllowsBoundaryValues(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	require.NoError(t, order.ApplyDiscount(0))
	require.Equal(t, 0.0, order.DiscountPercent)

	require.NoError(t, order.ApplyDiscount(100))
	require.Equal(t, 100.0, order.DiscountPercent)
}

func TestApplyDiscountReturnsErrorWhenOrderIsNotDraft(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusConfirmed

	err := order.ApplyDiscount(10)

	require.ErrorIs(t, err, ErrOrderIsNotDraft)
}

func TestApplyDiscountReturnsErrorWhenPercentIsNegative(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.ApplyDiscount(-1)

	require.ErrorIs(t, err, ErrInvalidDiscount)
}

func TestApplyDiscountReturnsErrorWhenPercentIsGreaterThanHundred(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.ApplyDiscount(101)

	require.ErrorIs(t, err, ErrInvalidDiscount)
}

func TestCalculateTotalReturnsZeroForEmptyOrder(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	total := order.CalculateTotal()

	require.Equal(t, 0.0, total)
}

func TestCalculateTotalAppliesDiscount(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Items["product-1"] = OrderItem{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 100,
	}
	order.Items["product-2"] = OrderItem{
		ProductID: "product-2",
		Quantity:  1,
		UnitPrice: 50,
	}
	order.DiscountPercent = 20

	total := order.CalculateTotal()

	require.Equal(t, 200.0, total)
}

func TestConfirmOrderConfirmsDraftOrder(t *testing.T) {
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 2).Return(true, nil).Once()

	order := NewOrder(inventory)
	order.Items["product-1"] = OrderItem{
		ProductID: "product-1",
		Quantity:  2,
		UnitPrice: 100,
	}

	err := order.ConfirmOrder()

	require.NoError(t, err)
	require.Equal(t, StatusConfirmed, order.Status)
	inventory.AssertExpectations(t)
}

func TestConfirmOrderReturnsErrorWhenOrderIsCancelled(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusCancelled

	err := order.ConfirmOrder()

	require.ErrorIs(t, err, ErrOrderIsCancelled)
	inventory.AssertNotCalled(t, "CheckAvailability")
}

func TestConfirmOrderReturnsErrorWhenOrderIsAlreadyConfirmed(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusConfirmed

	err := order.ConfirmOrder()

	require.ErrorIs(t, err, ErrOrderIsConfirmed)
	inventory.AssertNotCalled(t, "CheckAvailability")
}

func TestConfirmOrderReturnsErrorWhenOrderIsEmpty(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.ConfirmOrder()

	require.ErrorIs(t, err, ErrOrderIsEmpty)
	inventory.AssertNotCalled(t, "CheckAvailability")
}

func TestConfirmOrderReturnsInventoryError(t *testing.T) {
	inventoryErr := errors.New("inventory error")
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 1).Return(false, inventoryErr).Once()

	order := NewOrder(inventory)
	order.Items["product-1"] = OrderItem{
		ProductID: "product-1",
		Quantity:  1,
		UnitPrice: 100,
	}

	err := order.ConfirmOrder()

	require.ErrorIs(t, err, inventoryErr)
	require.Equal(t, StatusDraft, order.Status)
	inventory.AssertExpectations(t)
}

func TestConfirmOrderReturnsErrorWhenProductIsUnavailable(t *testing.T) {
	inventory := new(MockInventoryService)
	inventory.On("CheckAvailability", "product-1", 1).Return(false, nil).Once()

	order := NewOrder(inventory)
	order.Items["product-1"] = OrderItem{
		ProductID: "product-1",
		Quantity:  1,
		UnitPrice: 100,
	}

	err := order.ConfirmOrder()

	require.ErrorIs(t, err, ErrProductUnavailable)
	require.Equal(t, StatusDraft, order.Status)
	inventory.AssertExpectations(t)
}

func TestCancelOrderCancelsDraftOrder(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)

	err := order.CancelOrder()

	require.NoError(t, err)
	require.Equal(t, StatusCancelled, order.Status)
}

func TestCancelOrderReturnsErrorWhenOrderIsAlreadyCancelled(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusCancelled

	err := order.CancelOrder()

	require.ErrorIs(t, err, ErrOrderIsCancelled)
}

func TestCancelOrderReturnsErrorWhenOrderIsConfirmed(t *testing.T) {
	inventory := new(MockInventoryService)
	order := NewOrder(inventory)
	order.Status = StatusConfirmed

	err := order.CancelOrder()

	require.ErrorIs(t, err, ErrOrderIsConfirmed)
}
