package service

const (
	PermViewProject     = "view_project"
	PermManageProject   = "manage_project"
	PermManageMembers   = "manage_members"
	PermManageRoles     = "manage_roles"
	PermManageColumns   = "manage_columns"
	PermManageTasks     = "manage_tasks"
	PermManageStatuses  = "manage_statuses"
	PermManageTags      = "manage_tags"
	PermManageAssignees = "manage_assignees"
)

func AllPermissions() []string {
	return []string{
		PermViewProject, PermManageProject, PermManageMembers,
		PermManageRoles, PermManageColumns, PermManageTasks,
		PermManageStatuses, PermManageTags, PermManageAssignees,
	}
}

func IsValidPermission(name string) bool {
	for _, p := range AllPermissions() {
		if p == name {
			return true
		}
	}
	return false
}
