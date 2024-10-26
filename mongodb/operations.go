package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DbUser struct {
	Name     string `json:"name"`
	Password string `json:"password"`
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

type Privilege struct {
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
}

func (privilege Privilege) String() string {
	return fmt.Sprintf("{ resource : %s , actions : %s }", privilege.Resource, privilege.Actions)
}

type Resource struct {
	Db         string `json:"db"`
	Collection string `json:"collection"`
}

func (resource Resource) String() string {
	return fmt.Sprintf(" { db : %s , collection : %s }", resource.Db, resource.Collection)
}

type SingleResultGetUser struct {
	Users []struct {
		Id    string `json:"_id"`
		User  string `json:"user"`
		Db    string `json:"db"`
		Roles []struct {
			Role string `json:"role"`
			Db   string `json:"db"`
		} `json:"roles"`
	} `json:"users"`
}
type SingleResultGetRole struct {
	Roles []struct {
		Role           string `json:"role"`
		Db             string `json:"db"`
		InheritedRoles []struct {
			Role string `json:"role"`
			Db   string `json:"db"`
		} `json:"inheritedRoles"`
		Privileges []struct {
			Resource struct {
				Db         string `json:"db"`
				Collection string `json:"collection"`
			} `json:"resource"`
			Actions []string `json:"actions"`
		} `json:"privileges"`
	} `json:"roles"`
}

func createUser(client *mongo.Client, user DbUser, roles []Role, database string) error {
	var rolesValue interface{} = []bson.M{}
	if len(roles) != 0 {
		rolesValue = roles
	}

	result := client.Database(database).RunCommand(context.Background(), bson.D{
		{Key: "createUser", Value: user.Name},
		{Key: "pwd", Value: user.Password},
		{Key: "roles", Value: rolesValue},
	})

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getUser(client *mongo.Client, username string, database string) (SingleResultGetUser, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{
		Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: username},
			{Key: "db", Value: database},
		},
	}})
	var decodedResult SingleResultGetUser
	err := result.Decode(&decodedResult)
	if err != nil {
		return decodedResult, err
	}
	return decodedResult, nil
}

func dropUser(client *mongo.Client, username string, database string) error {
	result := client.Database(database).RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: username}})
	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getRole(client *mongo.Client, roleName string, database string) (SingleResultGetRole, error) {
	result := client.Database(database).RunCommand(context.Background(), bson.D{
		{Key: "rolesInfo", Value: bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: database}}},
		{Key: "showPrivileges", Value: true},
	})
	var decodedResult SingleResultGetRole
	err := result.Decode(&decodedResult)
	if err != nil {
		return decodedResult, err
	}
	return decodedResult, nil
}

func createRole(client *mongo.Client, role string, roles []Role, privilege []PrivilegeDto, database string) error {
	var privileges []Privilege
	for _, element := range privilege {
		var prv Privilege
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
