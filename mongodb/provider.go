package mongodb

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
			"ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_CA_FILE", ""),
				Description: "PEM-encoded content of Mongodb host CA certificate",
			},
			"certificate_key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("MONGODB_CERT_KEY_FILE", ""),
				Description: "PEM-encoded content of client certificate and key",
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
			"tls": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "tls activation",
			},
			"direct": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "enforces a direct connection instead of discovery",
			},
			"retrywrites": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Retryable Writes",
			},
			"proxy": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"ALL_PROXY",
					"all_proxy",
				}, nil),
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile(`^socks5h?://.*:\\d+$`), "The proxy URL is not a valid socks url.")),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"mongodb_db_user": resourceDatabaseUser(),
			"mongodb_db_role": resourceDatabaseRole(),
		},
		DataSourcesMap:       map[string]*schema.Resource{},
		ConfigureContextFunc: providerConfigure,
	}
}

type MongoDatabaseConfiguration struct {
	Config          *ClientConfig
	MaxConnLifetime time.Duration
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	clientConfig := ClientConfig{
		Host:               d.Get("host").(string),
		Port:               d.Get("port").(string),
		Username:           d.Get("username").(string),
		Password:           d.Get("password").(string),
		DB:                 d.Get("auth_database").(string),
		TLS:                d.Get("tls").(bool),
		ReplicaSet:         d.Get("replica_set").(string),
		CAFile:             d.Get("ca_file").(string),
		CertificateKeyFile: d.Get("certificate_key_file").(string),
		Direct:             d.Get("direct").(bool),
		RetryWrites:        d.Get("retrywrites").(bool),
		Proxy:              d.Get("proxy").(string),
	}

	return &MongoDatabaseConfiguration{
		Config:          &clientConfig,
		MaxConnLifetime: 10,
	}, diags

}
