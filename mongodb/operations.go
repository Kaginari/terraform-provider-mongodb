package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DbUser struct {
	AuthDatabase string          `json:"auth_database"`
	Name         string          `json:"name"`
	Password     string          `json:"password"`
	Roles        []RoleReference `json:"roles"`
}

type RoleReference struct {
	Role string `json:"role"`
	Db   string `json:"db"`
}

func (role RoleReference) String() string {
	return fmt.Sprintf("{ role : %s , db : %s }", role.Role, role.Db)
}

type Role struct {
	Name      string
	Database  string
	Roles     []RoleReference
	Privilege []Privilege
}

type Privilege struct {
	Db         string   `json:"db"`
	Collection string   `json:"collection"`
	Actions    []string `json:"actions"`
}

type MongodbPrivilege struct {
	Resource MongodbResource `json:"resource"`
	Actions  []string        `json:"actions"`
}

func (privilege MongodbPrivilege) String() string {
	return fmt.Sprintf("{ resource : %s , actions : %s }", privilege.Resource, privilege.Actions)
}

type MongodbResource struct {
	Db         string `json:"db"`
	Collection string `json:"collection"`
}

func (resource MongodbResource) String() string {
	return fmt.Sprintf(" { db : %s , collection : %s }", resource.Db, resource.Collection)
}

type MongodbRoleReference struct {
	Role string `json:"role"`
	Db   string `json:"db"`
}

type MongodbUser struct {
	Id    string                 `json:"_id"`
	User  string                 `json:"user"`
	Db    string                 `json:"db"`
	Roles []MongodbRoleReference `json:"roles"`
}

type MongodbUsersResult struct {
	Users []MongodbUser `json:"users"`
}

type MongodbRole struct {
	Role           string                 `json:"role"`
	Db             string                 `json:"db"`
	InheritedRoles []MongodbRoleReference `json:"inheritedRoles"`
	Privileges     []MongodbPrivilege     `json:"privileges"`
}

type MongodbRolesResult struct {
	Roles []MongodbRole `json:"roles"`
}

func createUser(client *mongo.Client, user *DbUser) error {
	var rolesValue interface{} = []bson.M{}
	if len(user.Roles) != 0 {
		rolesValue = user.Roles
	}

	result := client.Database(user.AuthDatabase).RunCommand(context.Background(), bson.D{
		{Key: "createUser", Value: user.Name},
		{Key: "pwd", Value: user.Password},
		{Key: "roles", Value: rolesValue},
	})

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getUser(client *mongo.Client, username string, database string, password string) (*DbUser, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{
		Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: username},
			{Key: "db", Value: database},
		},
	}})
	var decodedResult MongodbUsersResult
	err := result.Decode(&decodedResult)
	if err != nil {
		return nil, err
	}

	if len(decodedResult.Users) == 0 {
		return nil, fmt.Errorf("User %s.%s does not exist", database, username)
	}

	mongodbUser := decodedResult.Users[0]

	roles := make([]RoleReference, len(mongodbUser.Roles))
	for i, r := range mongodbUser.Roles {
		roles[i] = RoleReference{Role: r.Role, Db: r.Db}
	}

	user := DbUser{
		AuthDatabase: database,
		Name:         username,
		Password:     password,
		Roles:        roles,
	}

	return &user, nil
}

func dropUser(client *mongo.Client, username string, database string) error {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: username}})
	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getRole(client *mongo.Client, roleName string, database string) (*MongodbRole, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{
		{Key: "rolesInfo", Value: bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: database}}},
		{Key: "showPrivileges", Value: true},
	})
	var decodedResult MongodbRolesResult
	err := result.Decode(&decodedResult)
	if err != nil {
		return nil, err
	}

	if len(decodedResult.Roles) == 0 {
		return nil, fmt.Errorf("Role %s.%s does not exist", database, roleName)
	}

	return &decodedResult.Roles[0], nil
}

func createRole(client *mongo.Client, role *Role) error {
	var privilegesData []MongodbPrivilege
	for _, element := range role.Privilege {
		privilege := MongodbPrivilege{
			Resource: MongodbResource{
				Db:         element.Db,
				Collection: element.Collection,
			},
			Actions: element.Actions,
		}
		privilegesData = append(privilegesData, privilege)
	}

	var privilegesValue interface{} = []bson.M{}
	if len(privilegesData) != 0 {
		privilegesValue = privilegesData
	}

	var rolesValue interface{} = []bson.M{}
	if len(role.Roles) != 0 {
		rolesValue = role.Roles
	}

	result := client.Database(role.Database).RunCommand(context.Background(), bson.D{
		{Key: "createRole", Value: role.Name},
		{Key: "privileges", Value: privilegesValue},
		{Key: "roles", Value: rolesValue},
	})

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func dropRole(client *mongo.Client, role string, database string) error {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{Key: "dropRole", Value: role}})
	if result.Err() != nil {
		return result.Err()
	}
	return nil
}
