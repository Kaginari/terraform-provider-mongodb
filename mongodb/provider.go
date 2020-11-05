package mongodb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	var host = d.Get("host").(string)
	var port = d.Get("port").(string)
	var user = d.Get("username").(string)
	var pwd = d.Get("password").(string)
	var database = d.Get("auth_database").(string)

	var uri = "mongodb://" + host + ":" + port


	client, err := mongo.NewClient(options.Client().ApplyURI(uri).SetAuth(options.Credential{
		AuthSource: database, Username: user, Password: pwd,
	}))

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
