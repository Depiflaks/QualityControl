package subscription

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockBillingService struct {
	mock.Mock
}

func (m *MockBillingService) ChargeForUpgrade(newPlan string) error {
	args := m.Called(newPlan)
	return args.Error(0)
}

func TestNewSubscriptionPlanManagerSuccess(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)

	require.NoError(t, err)
	require.NotNil(t, manager)

	assert.Equal(t, PlanFree, manager.Plan)
	assert.Equal(t, 0, manager.UsedRequests)
}

func TestNewSubscriptionPlanManagerUnknownPlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager("premium", billing)

	require.Error(t, err)
	assert.Nil(t, manager)
}

func TestNewSubscriptionPlanManagerNilBilling(t *testing.T) {
	manager, err := NewSubscriptionPlanManager(PlanFree, nil)

	require.Error(t, err)
	assert.Nil(t, manager)
}

func TestUpgradePlanSuccess(t *testing.T) {
	billing := new(MockBillingService)
	billing.On("ChargeForUpgrade", PlanPro).Return(nil).Once()

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.UpgradePlan(PlanPro)

	require.NoError(t, err)
	assert.Equal(t, PlanPro, manager.Plan)
	billing.AssertExpectations(t)
}

func TestUpgradePlanToUnknownPlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.UpgradePlan("premium")

	require.Error(t, err)
	assert.Equal(t, PlanFree, manager.Plan)
	billing.AssertNotCalled(t, "ChargeForUpgrade", mock.Anything)
}

func TestUpgradePlanToSamePlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanPro, billing)
	require.NoError(t, err)

	err = manager.UpgradePlan(PlanPro)

	require.Error(t, err)
	assert.Equal(t, PlanPro, manager.Plan)
	billing.AssertNotCalled(t, "ChargeForUpgrade", mock.Anything)
}

func TestUpgradePlanToLowerPlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanBusiness, billing)
	require.NoError(t, err)

	err = manager.UpgradePlan(PlanPro)

	require.Error(t, err)
	assert.Equal(t, PlanBusiness, manager.Plan)
	billing.AssertNotCalled(t, "ChargeForUpgrade", mock.Anything)
}

func TestUpgradePlanBillingFailed(t *testing.T) {
	billing := new(MockBillingService)
	billing.On("ChargeForUpgrade", PlanPro).Return(errors.New("payment failed")).Once()

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.UpgradePlan(PlanPro)

	require.Error(t, err)
	assert.Equal(t, PlanFree, manager.Plan)
	billing.AssertExpectations(t)
}

func TestDowngradePlanSuccess(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanBusiness, billing)
	require.NoError(t, err)

	manager.UsedRequests = 500

	err = manager.DowngradePlan(PlanPro)

	require.NoError(t, err)
	assert.Equal(t, PlanPro, manager.Plan)
}

func TestDowngradePlanToUnknownPlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanBusiness, billing)
	require.NoError(t, err)

	err = manager.DowngradePlan("student")

	require.Error(t, err)
	assert.Equal(t, PlanBusiness, manager.Plan)
}

func TestDowngradePlanToSamePlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanPro, billing)
	require.NoError(t, err)

	err = manager.DowngradePlan(PlanPro)

	require.Error(t, err)
	assert.Equal(t, PlanPro, manager.Plan)
}

func TestDowngradePlanToHigherPlan(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.DowngradePlan(PlanPro)

	require.Error(t, err)
	assert.Equal(t, PlanFree, manager.Plan)
}

func TestDowngradePlanRejectedWhenUsedRequestsExceedNewLimit(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanPro, billing)
	require.NoError(t, err)

	manager.UsedRequests = 500

	err = manager.DowngradePlan(PlanFree)

	require.Error(t, err)
	assert.Equal(t, PlanPro, manager.Plan)
}

func TestConsumeRequestSuccess(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.ConsumeRequest(10)

	require.NoError(t, err)
	assert.Equal(t, 10, manager.UsedRequests)
}

func TestConsumeRequestSeveralTimes(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	require.NoError(t, manager.ConsumeRequest(10))
	require.NoError(t, manager.ConsumeRequest(15))

	assert.Equal(t, 25, manager.UsedRequests)
}

func TestConsumeRequestZeroCount(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.ConsumeRequest(0)

	require.Error(t, err)
	assert.Equal(t, 0, manager.UsedRequests)
}

func TestConsumeRequestNegativeCount(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.ConsumeRequest(-5)

	require.Error(t, err)
	assert.Equal(t, 0, manager.UsedRequests)
}

func TestConsumeRequestExceedsLimit(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.ConsumeRequest(101)

	require.Error(t, err)
	assert.Equal(t, 0, manager.UsedRequests)
}

func TestConsumeRequestExactlyLimit(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	err = manager.ConsumeRequest(100)

	require.NoError(t, err)
	assert.Equal(t, 100, manager.UsedRequests)
}

func TestCanUseFeature(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		feature  string
		expected bool
	}{
		{
			name:     "free plan can use basic feature",
			plan:     PlanFree,
			feature:  "basic",
			expected: true,
		},
		{
			name:     "pro plan can use basic feature",
			plan:     PlanPro,
			feature:  "basic",
			expected: true,
		},
		{
			name:     "business plan can use basic feature",
			plan:     PlanBusiness,
			feature:  "basic",
			expected: true,
		},
		{
			name:     "free plan cannot use analytics",
			plan:     PlanFree,
			feature:  "analytics",
			expected: false,
		},
		{
			name:     "pro plan can use analytics",
			plan:     PlanPro,
			feature:  "analytics",
			expected: true,
		},
		{
			name:     "business plan can use analytics",
			plan:     PlanBusiness,
			feature:  "analytics",
			expected: true,
		},
		{
			name:     "free plan cannot use team management",
			plan:     PlanFree,
			feature:  "team_management",
			expected: false,
		},
		{
			name:     "pro plan cannot use team management",
			plan:     PlanPro,
			feature:  "team_management",
			expected: false,
		},
		{
			name:     "business plan can use team management",
			plan:     PlanBusiness,
			feature:  "team_management",
			expected: true,
		},
		{
			name:     "business plan can use priority support",
			plan:     PlanBusiness,
			feature:  "priority_support",
			expected: true,
		},
		{
			name:     "pro plan cannot use priority support",
			plan:     PlanPro,
			feature:  "priority_support",
			expected: false,
		},
		{
			name:     "unknown feature is not available",
			plan:     PlanBusiness,
			feature:  "custom_feature",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			billing := new(MockBillingService)

			manager, err := NewSubscriptionPlanManager(tt.plan, billing)
			require.NoError(t, err)

			result := manager.CanUseFeature(tt.feature)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResetMonthlyUsage(t *testing.T) {
	billing := new(MockBillingService)

	manager, err := NewSubscriptionPlanManager(PlanFree, billing)
	require.NoError(t, err)

	manager.UsedRequests = 80

	manager.ResetMonthlyUsage()

	assert.Equal(t, 0, manager.UsedRequests)
}
