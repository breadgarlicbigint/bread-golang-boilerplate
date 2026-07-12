package service_test

import (
	"testing"

	svc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/service"
	tenantEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/entity"
)

func TestPlanMaxUsers(t *testing.T) {
	tests := []struct {
		plan tenantEntity.TenantPlan
		want int
	}{
		{tenantEntity.TenantPlanFree, 5},
		{tenantEntity.TenantPlanStarter, 5},
		{tenantEntity.TenantPlanPro, 50},
		{tenantEntity.TenantPlanEnterprise, 1000},
	}
	for _, tt := range tests {
		got := svc.ExportPlanMaxUsers(tt.plan)
		if got != tt.want {
			t.Errorf("planMaxUsers(%s) = %d; want %d", tt.plan, got, tt.want)
		}
	}
}
