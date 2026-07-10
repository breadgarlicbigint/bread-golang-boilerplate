package service

import "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/entity"

// ExportPlanMaxUsers exposes the private planMaxUsers for unit tests.
func ExportPlanMaxUsers(p entity.TenantPlan) int {
	return planMaxUsers(p)
}
