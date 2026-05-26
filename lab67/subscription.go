package subscription

import "errors"

const (
	PlanFree     = "free"
	PlanPro      = "pro"
	PlanBusiness = "business"
)

type BillingService interface {
	ChargeForUpgrade(newPlan string) error
}

type SubscriptionPlanManager struct {
	Plan         string
	UsedRequests int

	billing BillingService
}

var planLimits = map[string]int{
	PlanFree:     100,
	PlanPro:      1000,
	PlanBusiness: 10000,
}

var planRanks = map[string]int{
	PlanFree:     1,
	PlanPro:      2,
	PlanBusiness: 3,
}

func NewSubscriptionPlanManager(plan string, billing BillingService) (*SubscriptionPlanManager, error) {
	if _, ok := planLimits[plan]; !ok {
		return nil, errors.New("unknown subscription plan")
	}

	if billing == nil {
		return nil, errors.New("billing service is required")
	}

	return &SubscriptionPlanManager{
		Plan:         plan,
		UsedRequests: 0,
		billing:      billing,
	}, nil
}

func (s *SubscriptionPlanManager) UpgradePlan(newPlan string) error {
	if _, ok := planLimits[newPlan]; !ok {
		return errors.New("unknown subscription plan")
	}

	if planRanks[newPlan] <= planRanks[s.Plan] {
		return errors.New("new plan must be higher than current plan")
	}

	if err := s.billing.ChargeForUpgrade(newPlan); err != nil {
		return errors.New("upgrade payment failed")
	}

	s.Plan = newPlan
	return nil
}

func (s *SubscriptionPlanManager) DowngradePlan(newPlan string) error {
	if _, ok := planLimits[newPlan]; !ok {
		return errors.New("unknown subscription plan")
	}

	if planRanks[newPlan] >= planRanks[s.Plan] {
		return errors.New("new plan must be lower than current plan")
	}

	newLimit := planLimits[newPlan]

	if s.UsedRequests > newLimit {
		return errors.New("cannot downgrade: used requests exceed new plan limit")
	}

	s.Plan = newPlan
	return nil
}

func (s *SubscriptionPlanManager) ConsumeRequest(count int) error {
	if count <= 0 {
		return errors.New("request count must be positive")
	}

	limit := planLimits[s.Plan]

	if s.UsedRequests+count > limit {
		return errors.New("monthly request limit exceeded")
	}

	s.UsedRequests += count
	return nil
}

func (s *SubscriptionPlanManager) CanUseFeature(feature string) bool {
	switch feature {
	case "basic":
		return true
	case "analytics":
		return s.Plan == PlanPro || s.Plan == PlanBusiness
	case "team_management":
		return s.Plan == PlanBusiness
	case "priority_support":
		return s.Plan == PlanBusiness
	default:
		return false
	}
}

func (s *SubscriptionPlanManager) ResetMonthlyUsage() {
	s.UsedRequests = 0
}
