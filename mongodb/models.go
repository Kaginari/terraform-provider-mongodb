package mongodb

type User struct {
	AuthDatabase string
	Name         string
	Password     string
	Roles        []RoleReference
}

type RoleReference struct {
	Role string
	Db   string
}

type Role struct {
	Name       string
	Database   string
	Roles      []RoleReference
	Privileges []Privilege
}

type Privilege struct {
	Db         string
	Collection string
	Actions    []string
}
