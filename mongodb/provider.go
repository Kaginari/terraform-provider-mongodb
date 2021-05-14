package mongodb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGO_HOST", "127.0.0.1"),
				Description: "The mongodb server address",
			},
			"port": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGO_PORT", "27017"),
				Description: "The mongodb server port",
			},
			"ca_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_CA_MATERIAL", ""),
				Description: "PEM-encoded content of Mongodb host CA certificate",
			},
			"cert_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_CERT_MATERIAL", ""),
				Description: "PEM-encoded content of Mongodb client certificate",
			},
			"key_material": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_KEY_MATERIAL", ""),
				Description: "PEM-encoded content of Mongodb client private key",
			},

			"cert_path": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_CERT_PATH", ""),
				Description: "Path to directory with Mongodb TLS config",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGO_USR", nil),
				Description: "The mongodb user",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGO_PWD", nil),
				Description: "The mongodb password",
			},
			"auth_database": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "admin",
				Description: "The mongodb auth database",
			},
			"replica_set": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The mongodb replica set",
			},
			"ssl": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "ssl activation",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"mongodb_db_user": resourceDatabaseUser(),
			"mongodb_db_role": resourceDatabaseRole(),
		},
		DataSourcesMap: map[string]*schema.Resource{

		},
		ConfigureContextFunc: providerConfigure,

	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	clientConfig := ClientConfig{
		Host:     d.Get("host").(string),
		Port:     d.Get("port").(string),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
		DB:       d.Get("auth_database").(string),
		Ssl:      d.Get("ssl").(bool),
		ReplicaSet:      d.Get("replica_set").(string),
		Ca:       d.Get("ca_material").(string),
		Cert:     d.Get("cert_material").(string),
		Key:      d.Get("key_material").(string),
		CertPath: d.Get("cert_path").(string),
	}

	client, err := clientConfig.MongoClient()

	if err != nil {
		return nil, diag.Errorf("Error initializing Mongo connection %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		return nil, diag.Errorf("Error connecting to Mongo server %s", err)
	}
	err = client.Ping(ctx,nil)
	if err != nil {
		return nil, diag.Errorf("Error connecting to Mongo server %s", err)
	}
	return client,diags
}

