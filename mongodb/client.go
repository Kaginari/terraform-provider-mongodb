package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/proxy"
)

type ClientConfig struct {
	Host               string
	Port               string
	AuthSchema         string
	Username           string
	Password           string
	AuthX509Cert       string
	AuthX509Key        string
	DB                 string
	Ssl                bool
	InsecureSkipVerify bool
	ReplicaSet         string
	RetryWrites        bool
	Certificate        string
	Direct             bool
	Proxy              string
}

func addArgs(arguments string, newArg string) string {
	if arguments != "" {
		return arguments + "&" + newArg
	} else {
		return "/?" + newArg
	}

}

func (c *ClientConfig) MongoClient() (*mongo.Client, error) {
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

	dialer, dialerErr := proxyDialer(c)
	if dialerErr != nil {
		return nil, dialerErr
	}

	clientAuth, clientAuthErr := c.getClientAuth()
	if clientAuthErr != nil {
		return nil, clientAuthErr
	}

	tlsConfig, tlsConfigErr := c.getTlsConfig()
	if tlsConfigErr != nil {
		return nil, tlsConfigErr
	}

	clientOpts := options.Client().ApplyURI(uri).SetAuth(*clientAuth).SetDialer(dialer).SetTLSConfig(tlsConfig)

	client, err := mongo.NewClient(clientOpts)
	return client, err
}

func (c *ClientConfig) getClientAuth() (*options.Credential, error) {
	switch c.AuthSchema {
	case "PLAIN":
		if c.Username == "" || c.Password == "" {
			return nil, errors.New("The PLAIN auth schema requires user and password to be provided")
		}
		return &options.Credential{AuthSource: c.DB, Username: c.Username, Password: c.Password}, nil
	case "MONGODB-X509":
		if c.DB != "$external" {
			return nil, fmt.Errorf("The MONGODB-X509 auth schema requires the $external auth database, can't use %s", c.DB)
		}
		return &options.Credential{AuthMechanism: "MONGODB-X509", AuthSource: "$external"}, nil
	default:
		return nil, fmt.Errorf("Unknown auth schema: %s, should be PLAIN or MONGODB-X509", c.AuthSchema)
	}
}

func (c *ClientConfig) getTlsConfig() (*tls.Config, error) {
	if c.Certificate != "" || c.AuthX509Cert != "" {
		tlsConfig := new(tls.Config)
		tlsConfig.InsecureSkipVerify = c.InsecureSkipVerify
		if c.Certificate != "" {
			rootCertPool := x509.NewCertPool()
			ok := rootCertPool.AppendCertsFromPEM([]byte(c.Certificate))
			if !ok {
				return nil, errors.New("Failed parsing pem file")
			}
			tlsConfig.RootCAs = rootCertPool
		}
		if c.AuthX509Cert != "" {
			var privateKey = c.AuthX509Cert
			if c.AuthX509Key != "" {
				privateKey = c.AuthX509Key
			}
			clientCert, err := tls.LoadX509KeyPair(c.AuthX509Cert, privateKey)
			if err != nil {
				return nil, err
			}
			tlsConfig.GetClientCertificate = func(req *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &clientCert, nil
			}
		}
		return tlsConfig, nil
	}
	return nil, nil
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
