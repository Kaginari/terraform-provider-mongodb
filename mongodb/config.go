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
	"golang.org/x/net/proxy"
	"net/url"
	"strconv"
	"time"
)

type ClientConfig struct {
	Host               string
	Port               string
	Username           string
	Password           string
	DB                 string
	Ssl                bool
	InsecureSkipVerify bool
	ReplicaSet         string
	RetryWrites        bool
	Certificate        string
	CertificateKeyFile string
	Direct             bool
	Proxy              string
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
	Db         string   `json:"db"`
	Collection string   `json:"collection"`
	Actions    []string `json:"actions"`
}

type Privilege struct {
	Resource Resource `json:"resource"`
	Actions  []string `json:"actions"`
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

func addArgs(arguments string, newArg string) string {
	if arguments != "" {
		return arguments + "&" + newArg
	} else {
		return "/?" + newArg
	}

}

func (c *ClientConfig) MongoClient() (*mongo.Client, error) {
	dialer, dialerErr := proxyDialer(c)
	if dialerErr != nil {
		return nil, dialerErr
	}

	opts, err := c.clientOptions(dialer)
	if err != nil {
		return nil, err
	}

	return mongo.NewClient(opts)
}

// clientOptions builds the full *options.ClientOptions from the ClientConfig. Split out from
// MongoClient so the credential/TLS wiring can be unit-tested without a real network dialer or
// mongo.NewClient's own validation - a previous version of this logic built a *ClientOptions,
// called SetAuth on it, and then discarded the result in favor of a second, freshly-built one
// that never got the credential, so the auth mechanism it configured silently never took
// effect. Every credential path here is exercised by config_test.go against this function's
// actual return value, not just against whether MongoClient() returns an error.
func (c *ClientConfig) clientOptions(dialer options.ContextDialer) (*options.ClientOptions, error) {
	var verify = false
	var arguments = ""

	arguments = addArgs(arguments, "retrywrites="+strconv.FormatBool(c.RetryWrites))

	if c.Ssl {
		arguments = addArgs(arguments, "ssl=true")
	}

	if c.ReplicaSet != "" && c.Direct == false {
		arguments = addArgs(arguments, "replicaSet="+c.ReplicaSet)
	}

	if c.Direct {
		arguments = addArgs(arguments, "connect="+"direct")
	}

	var uri = "mongodb://" + c.Host + ":" + c.Port + arguments

	/*
		@Since: v0.0.9
		verify certificate
	*/
	if c.InsecureSkipVerify {
		verify = true
	}

	opts := options.Client().ApplyURI(uri).SetDialer(dialer)

	/*
		@Since: v0.1.8
		MONGODB-X509 client-certificate authentication: the certificate's subject DN is the
		identity, so no username/password is sent. Takes precedence over username/password
		when both happen to be set.
	*/
	if c.CertificateKeyFile != "" {
		opts = opts.SetAuth(options.Credential{AuthMechanism: "MONGODB-X509"})
	} else {
		opts = opts.SetAuth(options.Credential{
			AuthSource: c.DB, Username: c.Username, Password: c.Password,
		})
	}

	/*
		@Since: v0.0.7
		add certificate support for documentDB
	*/
	if c.Certificate != "" || c.CertificateKeyFile != "" {
		tlsConfig, err := getTLSConfigWithAllServerCertificates([]byte(c.Certificate), []byte(c.CertificateKeyFile), verify)
		if err != nil {
			return nil, err
		}
		opts = opts.SetTLSConfig(tlsConfig)
	}

	return opts, nil
}

func getTLSConfigWithAllServerCertificates(ca []byte, clientCertKeyPEM []byte, verify bool) (*tls.Config, error) {
	/* As of version 1.2.1, the MongoDB Go Driver will only use the first CA server certificate found in sslcertificateauthorityfile.
	   The code below addresses this limitation by manually appending all server certificates found in sslcertificateauthorityfile
	   to a custom TLS configuration used during client creation. */

	tlsConfig := new(tls.Config)

	tlsConfig.InsecureSkipVerify = verify

	if len(ca) > 0 {
		tlsConfig.RootCAs = x509.NewCertPool()
		ok := tlsConfig.RootCAs.AppendCertsFromPEM(ca)

		if !ok {
			return tlsConfig, errors.New("Failed parsing pem file")
		}
	}

	/*
		@Since: v0.1.8
		Client certificate + key for MONGODB-X509 auth, combined in one PEM value (mirrors
		how MongoDB's own docs distribute a single --tlsCertificateKeyFile). tls.X509KeyPair
		scans each argument for its own block type, so passing the same content twice is the
		documented way to parse a combined cert+key PEM.
	*/
	if len(clientCertKeyPEM) > 0 {
		clientCert, err := tls.X509KeyPair(clientCertKeyPEM, clientCertKeyPEM)
		if err != nil {
			return tlsConfig, fmt.Errorf("failed parsing client certificate/key pem file: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return tlsConfig, nil
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
	if len(roles) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: roles}})
	} else {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func updateUser(client *mongo.Client, user DbUser, roles []Role, database string) error {
	var result *mongo.SingleResult
	if len(roles) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: roles}})
	} else {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateUser", Value: user.Name},
			{Key: "pwd", Value: user.Password}, {Key: "roles", Value: []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func getUser(client *mongo.Client, username string, database string) (SingleResultGetUser, error) {
	var result *mongo.SingleResult
	result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "usersInfo", Value: bson.D{
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

func getRole(client *mongo.Client, roleName string, database string) (SingleResultGetRole, error) {
	var result *mongo.SingleResult
	result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "rolesInfo", Value: bson.D{
		{Key: "role", Value: roleName},
		{Key: "db", Value: database},
	},
	},
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
	var result *mongo.SingleResult
	for _, element := range privilege {
		var prv Privilege
		prv.Resource = Resource{
			Db:         element.Db,
			Collection: element.Collection,
		}
		prv.Actions = element.Actions
		privileges = append(privileges, prv)
	}
	if len(roles) != 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: roles}})
	} else if len(roles) == 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: []bson.M{}}})
	} else if len(roles) != 0 && len(privileges) == 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: roles}})
	} else {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "createRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func updateRole(client *mongo.Client, role string, roles []Role, privilege []PrivilegeDto, database string) error {
	var privileges []Privilege
	var result *mongo.SingleResult
	for _, element := range privilege {
		var prv Privilege
		prv.Resource = Resource{
			Db:         element.Db,
			Collection: element.Collection,
		}
		prv.Actions = element.Actions
		privileges = append(privileges, prv)
	}
	if len(roles) != 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: roles}})
	} else if len(roles) == 0 && len(privileges) != 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateRole", Value: role},
			{Key: "privileges", Value: privileges}, {Key: "roles", Value: []bson.M{}}})
	} else if len(roles) != 0 && len(privileges) == 0 {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: roles}})
	} else {
		result = client.Database(database).RunCommand(context.Background(), bson.D{{Key: "updateRole", Value: role},
			{Key: "privileges", Value: []bson.M{}}, {Key: "roles", Value: []bson.M{}}})
	}

	if result.Err() != nil {
		return result.Err()
	}
	return nil
}

func MongoClientInit(conf *MongoDatabaseConfiguration) (*mongo.Client, error) {

	client, err := conf.Config.MongoClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), conf.MaxConnLifetime*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func proxyDialer(c *ClientConfig) (options.ContextDialer, error) {
	proxyFromEnv := proxy.FromEnvironment().(options.ContextDialer)
	proxyFromProvider := c.Proxy

	if len(proxyFromProvider) > 0 {
		proxyURL, err := url.Parse(proxyFromProvider)
		if err != nil {
			return nil, err
		}
		proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return proxyDialer.(options.ContextDialer), nil
	}

	return proxyFromEnv, nil
}
