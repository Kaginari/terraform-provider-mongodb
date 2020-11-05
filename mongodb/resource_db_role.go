package mongodb

import (
	"context"
	"encoding/hex"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
)

func resourceDatabaseRole() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDatabaseRoleCreate,
		ReadContext:   resourceDatabaseRoleRead,
		UpdateContext: resourceDatabaseRoleUpdate,
		DeleteContext: resourceDatabaseRoleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: importDatabaseRoleState,
		},
		Schema: map[string]*schema.Schema{
			"database": {
				Type:     schema.TypeString,
				Optional: true,
				Default: "admin",
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"privilege": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"db": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"collection": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"actions": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"inherited_role": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 2,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceDatabaseRoleCreate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var client = i.(*mongo.Client)
	var role = data.Get("name").(string)
	var database = data.Get("database").(string)
	var roleList []Role
	var privileges []PrivilegeDto



	privilege := data.Get("privilege").(*schema.Set).List()
	roles := data.Get("inherited_role").(*schema.Set).List()

	mapstructure.Decode(roles, &roleList)
	mapstructure.Decode(privilege, &privileges)


	err := createRole(client, role, roleList, privileges, database)

	if err != nil {
		return diag.Errorf("Could not create the role : %s ", err)
	}
	str := database+"."+role
	hx := hex.EncodeToString([]byte(str))
	data.SetId(hx)
	return resourceDatabaseRoleRead(ctx, data, i)
}

func resourceDatabaseRoleDelete(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var client = i.(*mongo.Client)
	var stateId = data.State().ID
	id, errEncoding := hex.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}
	adminDB := client.Database("admin")
	Users := adminDB.Collection("system.roles")
	_, err := Users.DeleteOne(ctx, bson.M{"_id": string(id) })
	if err != nil {
		return diag.Errorf("%s",err)
	}

	return resourceDatabaseRoleRead(ctx, data, i)

}

func resourceDatabaseRoleUpdate(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var client = i.(*mongo.Client)
	var role = data.Get("name").(string)
	var database = data.Get("database").(string)
	var stateId = data.State().ID
	id, errEncoding := hex.DecodeString(stateId)
	if errEncoding != nil {
		return diag.Errorf("ID mismatch %s", errEncoding)
	}
	adminDB := client.Database("admin")
	Users := adminDB.Collection("system.roles")
	_, err := Users.DeleteOne(ctx, bson.M{"_id": string(id) })
	if err != nil {
		return diag.Errorf("%s",err)
	}
	var roleList []Role
	var privileges []PrivilegeDto

	privilege := data.Get("privilege").(*schema.Set).List()
	roles := data.Get("inherited_role").(*schema.Set).List()

	mapstructure.Decode(roles, &roleList)
	mapstructure.Decode(privilege, &privileges)

	err2 := createRole(client, role, roleList, privileges, database)

	if err2 != nil {
		return diag.Errorf("Could not create the role  :  %s ", err)
	}
	str := database+"."+role
	hx := hex.EncodeToString([]byte(str))
	data.SetId(hx)


	return resourceDatabaseRoleRead(ctx, data, i)
}

func resourceDatabaseRoleRead(ctx context.Context, data *schema.ResourceData, i interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	diags = nil
	return diags
}
func importDatabaseRoleState(ctx context.Context, data *schema.ResourceData, i interface{}) ([]*schema.ResourceData, error) {
	if err := data.Set("name", data.Get("name")); err != nil {
		return nil, err
	}
	data.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return []*schema.ResourceData{data}, nil
}
