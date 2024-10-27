package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DbUser struct {
	AuthDatabase string `json:"auth_database"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	Roles        []Role `json:"roles"`
}

type Role struct {
	Role string `json:"role"`
	Db   string `json:"db"`
}

func (role Role) String() string {
	return fmt.Sprintf("{ role : %s , db : %s }", role.Role, role.Db)
}

type PrivilegeDto struct {
	Db         string   `json:"db"`
	Collection string   `json:"collection"`
	Actions    []string `json:"actions"`
}

type MongodbPrivilege struct {
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
}

func (privilege MongodbPrivilege) String() string {
	return fmt.Sprintf("{ resource : %s , actions : %s }", privilege.Resource, privilege.Actions)
}

type Resource struct {
	Db         string `json:"db"`
	Collection string `json:"collection"`
}

func (resource Resource) String() string {
	return fmt.Sprintf(" { db : %s , collection : %s }", resource.Db, resource.Collection)
}

type MongodbUser struct {
	Id    string `json:"_id"`
	User  string `json:"user"`
	Db    string `json:"db"`
	Roles []Role `json:"roles"`
}

type MongodbUsersResult struct {
	Users []MongodbUser `json:"users"`
}

type MongodbRole struct {
	Role           string             `json:"role"`
	Db             string             `json:"db"`
	InheritedRoles []Role             `json:"inheritedRoles"`
	Privileges     []MongodbPrivilege `json:"privileges"`
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

	user := DbUser{
		AuthDatabase: database,
		Name:         username,
		Password:     password,
		Roles:        mongodbUser.Roles,
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

func createRole(client *mongo.Client, role string, roles []Role, privilege []PrivilegeDto, database string) error {
	var privileges []MongodbPrivilege
	for _, element := range privilege {
		var prv MongodbPrivilege
		prv.Resource = Resource{
			Db:         element.Db,
			Collection: element.Collection,
		}
		prv.Actions = element.Actions
		privileges = append(privileges, prv)
	}

	var privilegesValue interface{} = []bson.M{}
	if len(privileges) != 0 {
		privilegesValue = privileges
	}

	var rolesValue interface{} = []bson.M{}
	if len(roles) != 0 {
		rolesValue = roles
	}

	result := client.Database(database).RunCommand(context.Background(), bson.D{
		{Key: "createRole", Value: role},
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
