package mongodb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"strings"
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
	Db         string `json:"db"`
	Collection string `json:"collection"`
	Actions  []string `json:"actions"`
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


func createUser(client *mongo.Client, user DbUser, roles []Role, database string) error {
	var result *mongo.SingleResult
	if len(roles) != 0  {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createUser", user.Name},
			{"pwd", user.Password}, {"roles", roles}})
	} else{
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createUser", user.Name},
			{"pwd", user.Password}, {"roles", []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func createRole(client *mongo.Client, role string, roles []Role, privilege []PrivilegeDto, database string) error {
	var privileges []Privilege
	var result *mongo.SingleResult
	for _ , element := range privilege {
		var prv Privilege
		prv.Resource = Resource{
			Db:         element.Db,
			Collection: element.Collection,
		}
		prv.Actions = element.Actions
		privileges = append(privileges,prv)
	}
	if len(roles) != 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createRole", role},
			{"privileges", privileges}, {"roles", roles}})
	}else if len(roles) == 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createRole", role},
			{"privileges", privileges}, {"roles", []bson.M{}}})
	}else if len(roles) != 0 && len(privileges) == 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createRole", role},
			{"privileges", []bson.M{}}, {"roles", roles}})
	}else{
		result = client.Database(database).RunCommand(context.Background(), bson.D{{"createRole", role},
			{"privileges", []bson.M{}}, {"roles", []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}
func hex2int(hexStr string) uint64 {
	// remove 0x suffix if found in the input string
	cleaned := strings.Replace(hexStr, "0x", "", -1)

	// base 16 for hexadecimal
	result, _ := strconv.ParseUint(cleaned, 16, 64)
	return uint64(result)
}