package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


type ClientConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DB		 string
	Ssl      bool
	ReplicaSet string
	Ca       string
	Cert     string
	Key      string
	CertPath string
}
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
func prefixArgs(args string) string {
	if args != "" {
		return "&"
	} else {
		return "/?"
	}
}

func (c *ClientConfig) MongoClient() (*mongo.Client, error) {

	if c.Cert != "" || c.Key != "" {
		if c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("cert_material, and key_material must be specified")
		}

		if c.CertPath != "" {
			return nil, fmt.Errorf("cert_path must not be specified")
		}

		mongoClient, err := buildHTTPClientFromBytes([]byte(c.Ca), []byte(c.Cert), []byte(c.Key), c)
		if err != nil {
			return nil, err
		}
		return mongoClient,err
	}
	var arguments = ""
	if c.Ssl {
		arguments = prefixArgs(arguments)+"ssl=true"
	}
	if c.ReplicaSet != "" {
		arguments = prefixArgs(arguments)+"replicaSet="+c.ReplicaSet
	}
	var uri = "mongodb://" + c.Host + ":" + c.Port + arguments

	client, err := mongo.NewClient(options.Client().ApplyURI(uri).SetAuth(options.Credential{
		AuthSource: c.DB, Username: c.Username, Password: c.Password,
	}))
	return client, err
}
func buildHTTPClientFromBytes(caPEMCert, certPEMBlock, keyPEMBlock []byte, config *ClientConfig) (*mongo.Client, error) {
	tlsConfig := &tls.Config{}
	if certPEMBlock != nil && keyPEMBlock != nil {
		tlsCert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}

	if caPEMCert == nil || len(caPEMCert) == 0 {
		tlsConfig.InsecureSkipVerify = true
	} else {
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caPEMCert) {
			return nil, errors.New("Could not add RootCA pem")
		}
		tlsConfig.RootCAs = caPool
	}
	var arguments = ""
	if config.Ssl {
		arguments = prefixArgs(arguments)+"ssl=true"
	}
	if config.ReplicaSet != "" {
		arguments = prefixArgs(arguments)+"replicaSet="+config.ReplicaSet
	}
	var uri = "mongodb://" + config.Host + ":" + config.Port + arguments

	client, err := mongo.NewClient(options.Client().ApplyURI(uri).SetAuth(options.Credential{
			AuthSource: config.DB, Username: config.Username , Password: config.Password,
		}).SetTLSConfig(tlsConfig))

	return client , err
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
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: roles}})
	} else{
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: []bson.M{}}})
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
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: roles}})
	}else if len(roles) == 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: []bson.M{}}})
	}else if len(roles) != 0 && len(privileges) == 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: roles}})
	}else{
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}