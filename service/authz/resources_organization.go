package authz

const ResourceOrganization = "organization"

var (
	// OrganizationRead 查看部门列表与详情（含额度池与成员归属）
	OrganizationRead = Permission{Resource: ResourceOrganization, Action: ActionRead}
	// OrganizationWrite 新建/编辑部门、调整部门额度池
	OrganizationWrite = Permission{Resource: ResourceOrganization, Action: ActionWrite}
	// OrganizationMember 调整成员的部门归属
	OrganizationMember = Permission{Resource: ResourceOrganization, Action: "member"}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceOrganization,
		LabelKey: "Organization Management",
		Actions: []ActionDefinition{
			{
				Action:         ActionRead,
				LabelKey:       "Read organizations",
				DescriptionKey: "View organization list, quota pools, and member counts.",
				DefaultRoles:   []string{BuiltInRoleAdmin},
			},
			{
				Action:         ActionWrite,
				LabelKey:       "Edit organizations",
				DescriptionKey: "Create, edit, and delete organizations; adjust the quota pool.",
				DefaultRoles:   []string{BuiltInRoleAdmin},
			},
			{
				Action:         "member",
				LabelKey:       "Move members between organizations",
				DescriptionKey: "Change which organization a user belongs to.",
				DefaultRoles:   []string{BuiltInRoleAdmin},
			},
		},
	})
}
